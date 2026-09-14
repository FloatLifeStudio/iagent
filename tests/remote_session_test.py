"""阶段 0 远程全链路测试:单个 pexpect 会话内完成认证 + 上传 + 字段验证 + 推送验证。
用法:python3 tests/remote_session_test.py <TOTP>
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
    """等待目标机 shell 提示符(auth 完成后)。"""
    child.expect(PROMPT_MARKERS, timeout=timeout)


def run(child, cmd, timeout=120):
    """在会话内执行命令,返回输出。"""
    child.sendline(cmd)
    child.expect(PROMPT_MARKERS, timeout=timeout)
    return child.before


def send_file(child, local_path, remote_path, timeout=600):
    """base64 分块上传二进制文件。"""
    with open(local_path, "rb") as f:
        data = base64.b64encode(f.read()).decode()
    child.sendline(f"cat > {remote_path}.b64")
    # 4096 字符一行,减少往返
    for i in range(0, len(data), 4096):
        child.sendline(data[i:i + 4096])
    child.sendline("EOF")
    child.sendline(f"base64 -d {remote_path}.b64 > {remote_path} && chmod +x {remote_path} && rm {remote_path}.b64 && echo upload-ok")
    child.expect(["upload-ok", pexpect.TIMEOUT], timeout=timeout)


# --- 认证 ---
child = pexpect.spawn(
    f'ssh -p {PORT} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "{HOST}"',
    timeout=30, encoding="utf-8")
child.expect(["assword:", pexpect.TIMEOUT], timeout=30)
child.sendline(PASSWORD)
child.expect(["MFA", "OTP", "verification", "Verification", "code", pexpect.TIMEOUT], timeout=30)
child.sendline(OTP)

print("=== 认证完成,进入目标机 ===", flush=True)
wait_prompt(child)

# --- 上传 ---
print("=== 上传二进制(prototype 3.8MB,约 1-2 分钟) ===", flush=True)
run(child, f"mkdir -p {DIR}")
send_file(child, "dist/prototype", f"{DIR}/prototype")
print("prototype 上传完成", flush=True)

# --- 字段验证:非 root ---
print("=== 字段验证(非 root) ===", flush=True)
print(run(child, f"{DIR}/prototype 2>&1"), flush=True)

# --- 字段验证:root ---
print("=== 字段验证(root,sudo -S) ===", flush=True)
child.sendline(f"echo '{PASSWORD}' | sudo -S sh -c '{DIR}/prototype 2>&1'")
child.expect(PROMPT_MARKERS, timeout=120)
print(child.before, flush=True)

# --- 全链路推送验证:远端本机起哑服务,iagent 推给它(root 跑,验证 root 字段进 payload) ---
print("=== 全链路推送验证 ===", flush=True)
run(child, f"mkdir -p {DIR}")
run(child, "pkill -f 'http.server.*18080' || true")
run(child, "cat > /tmp/iagent-test/dummy.py << 'PYEOF'\n"
           "from http.server import BaseHTTPRequestHandler, HTTPServer\n"
           "class H(BaseHTTPRequestHandler):\n"
           "    def do_POST(self):\n"
           "        body = self.rfile.read(int(self.headers['Content-Length']))\n"
           "        open('/tmp/iagent-test/payload.json','wb').write(body)\n"
           "        self.send_response(200)\n"
           "        self.send_header('Content-Type','application/json')\n"
           "        self.end_headers()\n"
           "        self.wfile.write(b'{\"result\":\"created\",\"device_id\":1,\"pending_change_id\":null}')\n"
           "    def log_message(self, *a): pass\n"
           "HTTPServer(('127.0.0.1', 18080), H).handle_request()\n"
           "PYEOF")
run(child, "nohup python3 /tmp/iagent-test/dummy.py > /dev/null 2>&1 & sleep 1")
run(child, f"echo '{PASSWORD}' | sudo -S sh -c '{DIR}/iagent-0.1.0 --config /dev/null --server http://127.0.0.1:18080'")
child.sendline(f"echo '=== PAYLOAD ==='; cat /tmp/iagent-test/payload.json")
child.expect(PROMPT_MARKERS, timeout=60)
print(child.before, flush=True)

print("=== 测试完成 ===", flush=True)
child.sendline("exit")
child.close()
