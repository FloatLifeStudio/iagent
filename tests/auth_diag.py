"""一次性诊断:发送密码后打印服务器全部响应,不重试。"""
import pexpect
import sys

HOST = "JMS-e467e817-2d15-471b-bd69-e4b79df4f157@117.139.166.210"
PASSWORD = "DZNg0FbZCgloombq"

child = pexpect.spawn(
    f'ssh -p 49978 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "{HOST}"',
    timeout=15, encoding="utf-8")
child.expect(["assword:", pexpect.TIMEOUT], timeout=15)
child.sendline(PASSWORD)
# 只等待 10 秒,打印服务器全部响应,不做任何重试
try:
    child.expect([pexpect.TIMEOUT, pexpect.EOF], timeout=10)
except Exception:
    pass
print("=== 服务器响应原文 ===")
print(child.before)
print("=== 连接状态 ===")
print("closed:", child.closed)
child.close(force=True)
