"""Offline semantic-response retry contracts with synthetic HTTP replies only.

Inputs: source placeholders and provider doubles. Outputs: validation/retry
assertions. Effects: no external calls, database access or source mutations.
Choose this focused suite for semantic shape and sanitized feedback changes.
"""
import asyncio
import json
import sys
from pathlib import Path
from unittest.mock import AsyncMock, Mock

import httpx
import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import knowledge

VALID = {'summary': 'Explicit claim', 'classification': 'reference',
         'entities': [], 'statements': ['A supported statement.']}


def reply(value=VALID, finish_reason='stop', raw=False, status=200):
    """Build a synthetic HTTP response without contacting any provider.

    Inputs: semantic value, finish category, raw-content flag and HTTP status.
    Output: HTTPX response. Effects: none; use to exercise actual response checks.
    """
    return httpx.Response(status, request=httpx.Request('POST', 'https://provider.invalid/v1/chat/completions'),
                          json={'choices': [{'finish_reason': finish_reason,
                                             'message': {'content': value if raw else json.dumps(value)}}]})


@pytest.fixture
def provider(monkeypatch):
    """Install an in-memory provider and capture the existing request budget.

    Input: monkeypatch fixture. Output: client and sleep mocks. Effects: temporary
    test environment only; never reads real credentials or opens a network client.
    """
    for name, value in {'DOCSTORE_LLM_BASE_URL': 'https://provider.invalid/v1',
                        'DOCSTORE_LLM_MODEL': 'synthetic-model', 'DOCSTORE_LLM_API_KEY': 'synthetic',
                        'DOCSTORE_LLM_DISABLE_THINKING': '1'}.items():
        monkeypatch.setenv(name, value)
    client = AsyncMock()
    client.__aenter__.return_value = client
    monkeypatch.setattr(knowledge.httpx, 'AsyncClient', Mock(return_value=client))
    sleep = AsyncMock()
    monkeypatch.setattr(knowledge.asyncio, 'sleep', sleep)
    return client, sleep


def test_first_request_requires_explicit_arrays_without_retry_feedback(provider):
    """Specify the observed array contract on a successful first request.

    Inputs: valid synthetic response. Output: exact request/result assertions.
    Effects: mocks only; verifies provider settings and one-request success.
    """
    client, sleep = provider
    client.post.side_effect = [reply()]
    result = asyncio.run(knowledge.extract('source placeholder'))
    request = client.post.call_args.kwargs['json']
    assert request['messages'][0]['content'] == knowledge.SEMANTIC_PROMPT
    assert '{"summary":"","classification":"","entities":[],"statements":[]}' in knowledge.SEMANTIC_PROMPT
    assert 'JSON array of at most 8 strings' in knowledge.SEMANTIC_PROMPT
    assert request['max_tokens'] == 4096 and request['temperature'] == 0
    assert request['response_format'] == {'type': 'json_object'}
    assert request['chat_template_kwargs'] == {'enable_thinking': False}
    assert result['statements'] == VALID['statements'] and result['method'] == 'remote-llm'
    assert client.post.call_count == 1
    sleep.assert_not_called()


@pytest.mark.parametrize(('invalid', 'feedback'), [
    ({**VALID, 'statements': 'REJECTED_OUTPUT_MARKER'}, 'statements must be a JSON array of strings'),
    ({**VALID, 'entities': 'REJECTED_OUTPUT_MARKER'}, 'entities must be a JSON array of objects'),
    ({**VALID, 'summary': ['REJECTED_OUTPUT_MARKER']}, 'summary must be a string'),
    ({**VALID, 'classification': {'REJECTED_OUTPUT_MARKER': True}}, 'classification must be a string'),
    (['REJECTED_OUTPUT_MARKER'], 'wrong top-level type'),
])
def test_corrected_second_response_gets_only_sanitized_shape_feedback(provider, invalid, feedback):
    """Correct known field types without replaying rejected output or coercing it.

    Inputs: wrong-shaped response then valid response. Output: sanitized feedback
    and coverage assertions. Effects: mocks only; retains the existing two attempts.
    """
    client, _ = provider
    client.post.side_effect = [reply(invalid), reply()]
    result = asyncio.run(knowledge.extract('x' * 15000))
    first, second = [call.kwargs['json'] for call in client.post.call_args_list]
    assert first['messages'][0]['content'] == knowledge.SEMANTIC_PROMPT
    assert feedback in second['messages'][0]['content']
    assert 'REJECTED_OUTPUT_MARKER' not in json.dumps(second)
    assert [message['role'] for message in second['messages']] == ['system', 'user']
    assert len(first['messages'][1]['content']) == 12000
    assert len(second['messages'][1]['content']) == 6000
    assert result['statements'] == VALID['statements']
    assert result['coverage'] == {'source_chars': 15000, 'processed_chars': 6000, 'truncated': True}
    assert client.post.call_count == 2


@pytest.mark.parametrize(('response', 'feedback'), [
    (reply('REJECTED_OUTPUT_MARKER invalid JSON', raw=True), 'not valid JSON'),
    (reply(None, raw=True), 'not valid JSON'),
    (reply(VALID, finish_reason='length'), 'was truncated'),
])
def test_json_or_truncation_rejection_informs_second_attempt(provider, response, feedback):
    """Preserve JSON and length rejection while describing only its safe category.

    Inputs: rejected response then valid response. Output: accepted second result
    and category assertion. Effects: mocks only; never repairs truncated JSON.
    """
    client, _ = provider
    client.post.side_effect = [response, reply()]
    assert asyncio.run(knowledge.extract('source placeholder'))['method'] == 'remote-llm'
    second = client.post.call_args_list[1].kwargs['json']
    assert feedback in second['messages'][0]['content']
    assert 'REJECTED_OUTPUT_MARKER' not in json.dumps(second)


@pytest.mark.parametrize('response', [reply({**VALID, 'statements': 'scalar'}),
                                    reply('invalid', raw=True), reply(VALID, finish_reason='length')])
def test_two_malformed_responses_fail_without_fallback_or_third_request(provider, response):
    """Fail closed after two rejected replies rather than accepting scalar claims.

    Inputs: two malformed responses. Output: ValueError and two-request bound.
    Effects: mocks only; verifies no deterministic fallback or extra provider call.
    """
    client, _ = provider
    client.post.side_effect = [response, response]
    with pytest.raises(ValueError, match='semantic JSON twice'):
        asyncio.run(knowledge.extract('source placeholder'))
    assert client.post.call_count == 2


@pytest.mark.parametrize(('failure', 'delay'), [
    (httpx.ConnectError('synthetic transport failure'), 2),
    (reply(status=429), 5), (reply(status=503), 5),
])
def test_transport_and_retryable_http_behavior_retains_shared_budget(provider, failure, delay):
    """Keep transport/status retries within the existing two-request budget.

    Inputs: transient failure then valid response. Output: delay and unchanged
    prompt assertions. Effects: mocks only; no semantic feedback for HTTP failure.
    """
    client, sleep = provider
    client.post.side_effect = [failure, reply()]
    assert asyncio.run(knowledge.extract('source placeholder'))['method'] == 'remote-llm'
    assert client.post.call_count == 2
    assert client.post.call_args_list[1].kwargs['json']['messages'][0]['content'] == knowledge.SEMANTIC_PROMPT
    sleep.assert_awaited_once_with(delay)


def test_http_failure_does_not_gain_an_extra_attempt_after_shape_rejection(provider):
    """Keep the second attempt terminal when correction encounters a provider error.

    Inputs: scalar statements then HTTP 503. Output: propagated HTTPStatusError.
    Effects: mocks only; proves shape and transport retries share one bounded loop.
    """
    client, sleep = provider
    client.post.side_effect = [reply({**VALID, 'statements': 'scalar'}), reply(status=503)]
    with pytest.raises(httpx.HTTPStatusError):
        asyncio.run(knowledge.extract('source placeholder'))
    assert client.post.call_count == 2
    sleep.assert_not_called()
