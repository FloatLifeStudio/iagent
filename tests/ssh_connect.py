"""Phase 0 connection helper: use pexpect to handle JumpServer password + OTP
multi-step prompts, establish a ControlMaster main connection."""
import pexpect
import sys

CMD = ('ssh -p 49978 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null '
       '-o ControlMaster=auto -o ControlPath=/tmp/iagent-ssh-%r@%h:%p -o ControlPersist=4h '
       '-fN "Boris.Chen#ictrek#fe5c617e-8cf4-4a1d-8769-b45c18467e07@117.139.166.210"')
PASSWORD = 'o689ia97ic624d23'
OTP = sys.argv[1] if len(sys.argv) > 1 else ''

child = pexpect.spawn(CMD, timeout=25, encoding='utf-8')
prompts = ['assword:', 'MFA', 'OTP', 'OTP code', 'verification', 'Verification', 'code:', pexpect.EOF]
password_sent = False
otp_sent = False
while True:
    i = child.expect(prompts, timeout=25)
    matched = prompts[i]
    if matched == pexpect.EOF:
        break
    if matched == 'assword:' and not password_sent:
        child.sendline(PASSWORD)
        password_sent = True
    elif not otp_sent:
        child.sendline(OTP)
        otp_sent = True
    else:
        break
print('--- ssh output ---')
print(child.before)
