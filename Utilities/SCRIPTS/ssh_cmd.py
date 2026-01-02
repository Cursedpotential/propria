import subprocess
import sys

cmd = sys.argv[1] if len(sys.argv) > 1 else "hostname"
result = subprocess.run(
    ["ssh", "-o", "StrictHostKeyChecking=no", "root@172.233.222.234", cmd],
    capture_output=True,
    text=True,
    timeout=30
)
print("STDOUT:", result.stdout)
print("STDERR:", result.stderr)
print("EXIT:", result.returncode)
