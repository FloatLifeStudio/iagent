"""Phase 0 remote end-to-end test: auth + upload + field verification + push
verification within a single pexpect session.
Usage: python3 tests/remote_session_test.py <TOTP>
"""
import base64
import pexpect
import sys
import time

HOST = "Boris.Chen#ictrek#fe5c617e-8cf4-4a1d-8769-b45c18467e07@117.139.166.210"
PORT = "49978"
PASSWORD = "o689ia97ic624d23"
OTP = sys.argv[1] if len(sys.argv) > 1 else ""
DIR = "/tmp/iagent-test"

PROMPT_MARKERS = [r"oneshot-auth-done", r"\$ $", r"# $"]


def wait_prompt(child, timeout=60):
    """Wait for the target shell prompt (after auth)."""
    child.expect(PROMPT_MARKERS, timeout=timeout)


def run(child, cmd, timeout=120):
    """Execute a command in the session, return the output."""
    child.sendline(cmd)
    child.expect(PROMPT_MARKERS, timeout=timeout)
    return child.before


def send_file(child, local_path, remote_path, timeout=600):
    """Upload a binary file in base64 chunks."""
    with open(local_path, "rb") as f:
        data = base64.b64encode(f.read()).decode()
    child.sendline(f"cat > {remote_path}.b64")
    # 4096 chars per line to reduce round trips
    for i in range(0, len(data), 4096):
        child.sendline(data[i:i + 4096])
    child.sendline("EOF")
    child.sendline(f"base64 -d {remote_path}.b64 > {remote_path} && chmod +x {remote_path} && rm {remote_path}.b64 && echo upload-ok")
    child.expect(["upload-ok", pexpect.TIMEOUT], timeout=timeout)


# --- auth ---
child = pexpect.spawn(
    f'ssh -p {PORT} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "{HOST}"',
    timeout=30, encoding="utf-8")
child.expect(["assword:", pexpect.TIMEOUT], timeout=30)
child.sendline(PASSWORD)
child.expect(["MFA", "OTP", "verification", "Verification", "code", pexpect.TIMEOUT], timeout=30)
child.sendline(OTP)

print("=== auth done, entering target machine ===", flush=True)
wait_prompt(child)

# --- upload ---
print("=== uploading binary (prototype 3.8MB, ~1-2 minutes) ===", flush=True)
run(child, f"mkdir -p {DIR}")
send_file(child, "dist/prototype", f"{DIR}/prototype")
print("prototype upload done", flush=True)

# --- field verification: non-root ---
print("=== field verification (non-root) ===", flush=True)
print(run(child, f"{DIR}/prototype 2>&1"), flush=True)

# --- field verification: root ---
print("=== field verification (root, sudo -S) ===", flush=True)
child.sendline(f"echo '{PASSWORD}' | sudo -S sh -c '{DIR}/prototype 2>&1'")
child.expect(PROMPT_MARKERS, timeout=120)
print(child.before, flush=True)
