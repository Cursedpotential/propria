# Byline: Claude Code · Fable 5.1 · 2026-09-21 — start one TEST-mode proffer run through the Workbench API.
# usage: proffer_test_run_start.py <request_id> <source_ref> <declared_format>
# Prints the preview handle. Drive the gates with proffer_test_run_driver.py.
# TEST matter / court case are the fixed disposable pair used by every test run.
import json
import sys
import urllib.request

BASE = "https://workbench.tilapia-skilift.ts.net/api/proffer"
request_id, source_ref, declared_format = sys.argv[1], sys.argv[2], sys.argv[3]
body = {
    "request_id": request_id,
    "matter_id": "deadbeef-dead-beef-dead-beefdeadbeef",
    "court_case_id": "cafebabe-cafe-babe-cafe-babecafebabe",
    "source_ref": source_ref,
    "declared_format": declared_format,
    "parser_options_ref": "pending-handler-selection/v1",
    "matter_mode": "TEST",
}
req = urllib.request.Request(
    f"{BASE}/start?mode=TEST",
    data=json.dumps(body).encode(),
    headers={"content-type": "application/json"},
    method="POST",
)
try:
    response = json.load(urllib.request.urlopen(req, timeout=90))
except urllib.error.HTTPError as error:
    print(error.code, error.read().decode()[:600])
    raise SystemExit(1)
print(response["preview_handle"])
