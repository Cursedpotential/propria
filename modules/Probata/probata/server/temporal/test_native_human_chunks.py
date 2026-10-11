"""Verify native chunk receipt provenance with synthetic SQL/SDK coordinates only."""
import json
import unittest
from datetime import datetime, timedelta, timezone
from types import SimpleNamespace

from server.temporal.native_human_chunks import persist_chunk_receipt


class ReceiptTests(unittest.TestCase):
    """Keep actual retry attempt/start separate from completion and persistence clocks."""
    def test_actual_retry_clock_and_real_receipt_insert(self):
        """The fourth attempt retains its SDK start and inserts the returned UUID receipt."""
        request = {"workflow_id": "synthetic-workflow", "manifest_ref": "file:///synthetic/extracted.json",
                   "source": {"source_version_ref": "11111111-1111-4111-8111-111111111111", "source_ref": "synthetic://sms.xml"}}
        result = {"contract_version": "human-source-v1", "stage": "chunked", "manifest_ref": "file:///synthetic/chunked.json"}
        started = datetime(2026, 10, 11, 1, 2, 3, tzinfo=timezone.utc)
        completed = started + timedelta(minutes=7)
        info = SimpleNamespace(started_time=started, attempt=4, workflow_id="synthetic-workflow", workflow_run_id="synthetic-run", activity_id="synthetic-chunk")
        calls = []
        class Connection:
            """Record SQL parameters and provide synthetic registration/execution rows."""
            def __enter__(self): return self
            def __exit__(self, *args): return False
            def execute(self, statement, values):
                sql = str(statement)
                calls.append((sql, values))
                class Row:
                    def scalar(self): return None
                    def one(self):
                        return ("synthetic-workflow", "synthetic://sms.xml") if "JOIN context.source" in sql else ("22222222-2222-4222-8222-222222222222", "synthetic-workflow")
                return Row()
        engine = SimpleNamespace(begin=lambda: Connection())
        saved = persist_chunk_receipt(engine, request, result, info, completed_at=completed)
        insert = next(values for sql, values in calls if "INSERT INTO context.activity_receipt" in sql)
        self.assertEqual(insert["id"], saved["receipt_ref"])
        self.assertEqual(insert["attempt"], 4)
        self.assertEqual(insert["started"], started)
        self.assertEqual(insert["completed"], completed)
        self.assertNotEqual(insert["started"], insert["completed"])
        metadata = json.loads(next(values["metadata"] for sql, values in calls if "INSERT INTO context.source_metadata" in sql))
        self.assertEqual(metadata["temporal"]["workflow_run_id"], "synthetic-run")
        self.assertEqual(metadata["temporal"]["started_at"], started.isoformat())
        self.assertEqual(metadata["result"]["receipt_ref"], saved["receipt_ref"])
        self.assertFalse(any("CREATE TABLE" in sql or "normalized_record" in sql for sql, _ in calls))


if __name__ == "__main__":
    unittest.main()
