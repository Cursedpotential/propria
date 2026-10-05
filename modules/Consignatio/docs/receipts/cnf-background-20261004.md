# CNF background consolidation repair

> _Byline: Codex · GPT-6 · 2026-10-04, with Implementation agent GPT-6.1._

STATUS: VERIFIED — installed Codex dispatcher returned `{}` in 222 ms; manually forced dispatcher returned in 353 ms and background Ollama Cloud classification completed. Automatic threshold is 20 entries, no clock cooldown or growth gate; singleton OS lock, recursive guard and failure fingerprint suppress duplicate requests. Worker output is at most six groups without rewriting original memory contents.

Live status: {"status": "complete", "pid": 38364, "recorded_at": "2026-10-05T02:09:47.469043+00:00", "snapshot_count": 8, "group_count": 6, "concurrent_appends": 0, "backup": "E:\\AI_Workspace\\Projects\\Propria\\to_be_deleted\\cnf-recall-cleanup\\project_memory-38de6382dfea47dda58725f6d7e542d2.json", "model": "glm-5.2:cloud", "provider": "ollama_cloud", "snapshot_fingerprint": "d01493ae14d49f304a2f4eb7eba2f27a4ad2e5e65963d94f063f94981db5025b"}

Independent validation: all 9 original records from the immediate pre-run backup remain exact as JSON records beneath the groups; manual memories and other substantive document keys are unchanged. Captures after a snapshot are retained and can advance `updated_at`. An earlier hosted Claude probe failed OAuth refresh; the working default is Ollama Cloud `glm-5.2:cloud`, with no local-model fallback.

Source: `E:/AI_Workspace/plugins/plugins/claude-never-forgets/hooks/cleanup_worker.py` SHA-256 `67984b2a7f24df07e899ebea0d2cedda8c86ecaa87557e32c0ad3eebe12e5823`; `stop_cleanup.py` dispatch, `storage.py` locks/atomic writes, `user_prompt.py` complete prompt/exact duplicate capture, `tool_rejected.py` complete guarded feedback, `session_start.py` retained-topic recall. Both app caches use source copies. Synthetic worker suite: 21 tests passed; parent checks verify retained topics with accumulated captures, rejection feedback over 300 characters and the six-group bound. Fixtures and original source backups remain under `to_be_deleted/`; no permanent deletion or host lifecycle action.
