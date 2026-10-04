"""Screenshot-to-message matching: corroborate existing records, flag the rest, create nothing.

Byline: Claude Code · Sonnet 5.5 · 2026-10-03. Synthetic messages and OCR text only.
"""

from datetime import UTC, datetime, timedelta

from casebible_index.screenshot_match import (
    METHOD,
    MessageRef,
    match_screenshot,
    narrow_candidates,
    normalize,
    ocr_lines,
)

T = datetime(2026, 1, 2, 15, 0, tzinfo=UTC)
MESSAGES = [
    MessageRef(
        "r1",
        "Can you pick up Mia from school on Friday at 4:30?",
        T,
        "+18105550101",
        ("+18105550199",),
    ),
    MessageRef(
        "r2",
        "Yes, I will be at the front gate.",
        T + timedelta(minutes=2),
        "+18105550199",
        ("+18105550101",),
    ),
    MessageRef(
        "r3",
        "Dentist moved to Tuesday 9am",
        T + timedelta(days=30),
        "+18105550101",
        ("+18105550199",),
    ),
    MessageRef("r4", "unrelated lunch plans tomorrow", T, "+13135550000", ("+18105550101",)),
]
OCR = """Dana
Can you pick up Mia from
school on Friday at 4:30?
9:04 AM
Yes, | will be at the front gate.
Delivered
I never agreed to pay the doctor bill this month
"""


def test_normalization_and_chrome_removal():
    assert normalize("  Pick-up  AT 4:30!! ") == "pick-up at 4:30"
    texts = [line.text for line in ocr_lines(OCR)]
    assert "9:04 AM" not in texts and "Delivered" not in texts and "Dana" in texts


def test_wrapped_bubble_and_ocr_noise_match_their_records_and_the_rest_is_flagged():
    result = match_screenshot(OCR, MESSAGES)
    assert result.method == METHOD
    by = {m.record_id: m for m in result.matches}
    assert set(by) == {"r1", "r2"}, "r3 is a different message and r4 is unrelated"
    assert by["r1"].score >= 0.95 and by["r2"].score >= 0.82
    first = OCR[by["r1"].span[0] : by["r1"].span[1]]
    assert first.startswith("Can you pick up Mia") and first.endswith("4:30?"), (
        "the span is the OCR text matched"
    )
    flagged = [p.text for p in result.unmatched]
    assert flagged == ["I never agreed to pay the doctor bill this month"], (
        "text with no record is flagged, not created"
    )


def test_a_record_is_used_once_and_short_passages_prove_nothing():
    result = match_screenshot(
        "ok\nCan you pick up Mia from school on Friday at 4:30?\n"
        "Can you pick up Mia from school on Friday at 4:30?",
        MESSAGES[:1],
    )
    assert (
        len(result.matches) == 1
        and result.unmatched
        and result.unmatched[0].text.startswith("Can you pick up")
    )
    assert all(p.text != "ok" for p in result.unmatched)


def test_no_candidates_flags_every_substantial_passage():
    result = match_screenshot("Pick up Mia at 4:30 on Friday", [])
    assert result.matches == [] and [p.text for p in result.unmatched] == [
        "Pick up Mia at 4:30 on Friday"
    ]


def test_candidates_are_narrowed_by_capture_time_and_visible_parties():
    near = narrow_candidates(MESSAGES, capture_time=T + timedelta(hours=1))
    assert [m.record_id for m in near] == ["r1", "r2", "r4"], "r3 is a month away"
    by_number = narrow_candidates(MESSAGES, capture_time=T, visible_names=("(810) 555-0199",))
    assert [m.record_id for m in by_number] == ["r1", "r2"], "r4 involves neither visible party"
    assert [m.record_id for m in narrow_candidates(MESSAGES, visible_names=("Dana",))] == [], (
        "no record names Dana"
    )
    undated = MessageRef("r5", "x" * 20)
    assert narrow_candidates([undated], capture_time=T) == [undated], (
        "no time is not evidence of a different time"
    )
