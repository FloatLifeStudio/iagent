"""持久 SSH 会话保持器:认证后常驻,通过 FIFO 接收命令,在会话内执行并输出到文件。

用法:python3 tests/session_holder.py <TOTP>
驱动:echo '<cmd>' > /tmp/iagent-ssh-cmd.fifo ;输出追加到 /tmp/iagent-ssh-out.txt
特殊命令:
  UPLOAD:<local_path>:<remote_path>   — 上传本地文件(base64 分块)
  ___EXIT___                          — 退出会话
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

# FIFO 由脚本自行创建,避免启动前残留/缺失问题
if os.path.exists(FIFO):
    os.unlink(FIFO)
os.mkfifo(FIFO)


def log(msg):
    with open(OUT, "a") as f:
        f.write(str(msg) + "\n")


# --- 认证(免 TOTP,仅密码) ---
child = pexpect.spawn(
    f'ssh -p {PORT} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null '
    f'-o ExitOnForwardFailure=no -R 18080:127.0.0.1:18080 "{HOST}"',
    timeout=30, encoding="utf-8")
child.expect(["assword:", pexpect.TIMEOUT], timeout=30)
child.sendline(PASSWORD)
log("auth: password sent")

# --- 等目标机 shell,设置唯一提示符 ---
child.expect([r"\$ $", r"# $"], timeout=60)
child.sendline("export PS1='" + PROMPT + "'")
child.sendline("echo AUTH-READY")
child.expect("AUTH-READY", timeout=30)
child.expect(PROMPT)
log("auth: session ready")

# --- 命令循环 ---
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
                # pty 行缓冲上限 4096,分块必须 ≤4000 字符(超过会永久死锁)
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
            # 普通命令:执行,输出到 OUT
            child.sendline(cmd)
            child.expect(PROMPT, timeout=300)
            log(child.before)
