"""Persistent SSH session holder: stays alive after auth, receives commands via a
FIFO, executes them in the session and writes output to a file.

Usage: python3 tests/session_holder.py <TOTP>
Drive: echo '<cmd>' > /tmp/iagent-ssh-cmd.fifo ; output appended to /tmp/iagent-ssh-out.txt
Special commands:
  UPLOAD:<local_path>:<remote_path>   -- upload a local file (base64 chunks)
  ___EXIT___                          -- exit the session
"""
import base64
import os
import pexpect
import sys

HOST = "JMS-52b31fbf-91ec-4499-8ead-d5696bb4b65a@117.139.166.210"
PORT = "49978"
PASSWORD = "j33Zt1mFxi6re3NW"
OTP = sys.argv[1] if len(sys.argv) > 1 else ""
FIFO = "/tmp/iagent-ssh-cmd.fifo"
OUT = "/tmp/iagent-ssh-out.txt"
PROMPT = "IAGENT_PROMPT_"

# the FIFO is created by this script itself, avoiding stale/missing issues before startup
if os.path.exists(FIFO):
    os.unlink(FIFO)
os.mkfifo(FIFO)


def log(msg):
    with open(OUT, "a") as f:
        f.write(str(msg) + "\n")


# --- auth (no TOTP, password only) ---
child = pexpect.spawn(
    f'ssh -p {PORT} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null '
    f'-o ExitOnForwardFailure=no -R 18080:127.0.0.1:18080 "{HOST}"',
    timeout=30, encoding="utf-8")
child.expect(["assword:", pexpect.TIMEOUT], timeout=30)
child.sendline(PASSWORD)
log("auth: password sent")

# --- wait for the target shell, set a unique prompt marker ---
child.expect([r"\$ $", r"# $"], timeout=60)
child.sendline("export PS1='" + PROMPT + "'")
child.sendline("echo AUTH-READY")
child.expect("AUTH-READY", timeout=30)
child.expect(PROMPT)
log("auth: session ready")

# --- command loop ---
while True:
    with open(FIFO) as fifo:
        for line in fifo:
            cmd = line.rstrip("\n")
            if not cmd:
                continue
            if cmd == "___EXIT___":
                child.sendline("exit")
                child.close()
                sys.exit(0)
            if cmd.startswith("UPLOAD:"):
                _, local, remote = cmd.split(":", 2)
                with open(local, "rb") as f:
                    data = base64.b64encode(f.read()).decode()
                child.sendline(f"cat > {remote}.b64")
                # pty line buffer limit is 4096, chunks must be <= 4000 chars
                # (larger chunks deadlock permanently)
                n = 0
                for i in range(0, len(data), 4000):
                    child.sendline(data[i:i + 4000])
                    n += 1
                    if n % 200 == 0:
                        log(f"  upload {remote}: {n} chunks / {len(data)//4000}")
                child.sendline("EOF")
                child.sendline(f"base64 -d {remote}.b64 > {remote} && chmod +x {remote} && rm {remote}.b64 && echo UPLOAD-OK")
                child.expect("UPLOAD-OK", timeout=600)
                child.expect(PROMPT, timeout=60)
                log(f"upload {remote}: ok")
                continue
            # normal command: execute, output to OUT
            child.sendline(cmd)
            child.expect(PROMPT, timeout=300)
            log(child.before)
