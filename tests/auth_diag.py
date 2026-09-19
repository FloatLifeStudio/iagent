"""One-shot diagnosis: send the password and print the full server response, no retry."""
import pexpect
import sys

HOST = "JMS-e467e817-2d15-471b-bd69-e4b79df4f157@117.139.166.210"
PASSWORD = "DZNg0FbZCgloombq"

child = pexpect.spawn(
    f'ssh -p 49978 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "{HOST}"',
    timeout=15, encoding="utf-8")
child.expect(["assword:", pexpect.TIMEOUT], timeout=15)
child.sendline(PASSWORD)
# wait 10 seconds only, print the full server response, no retry at all
try:
    child.expect([pexpect.TIMEOUT, pexpect.EOF], timeout=10)
except Exception:
    pass
print("=== raw server response ===")
print(child.before)
print("=== connection status ===")
print("closed:", child.closed)
child.close(force=True)
