"""Local helper server (runs on 23-HCI):
- GET  /iagent.gz          -> serves the stripped binary (via SSH reverse tunnel to remote)
- POST /api/v1/devices     -> captures the payload pushed by remote, saves to file
- GET  /health             -> health check
"""
import json
from http.server import BaseHTTPRequestHandler, HTTPServer

BIN = "/data/iagent/dist/iagent-0.1.0.gz"
PAYLOAD = "/data/iagent/tests/remote_payload.json"


class H(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/iagent.gz":
            with open(BIN, "rb") as f:
                data = f.read()
            self.send_response(200)
            self.send_header("Content-Type", "application/octet-stream")
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            self.wfile.write(data)
        elif self.path == "/health":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        if self.path == "/api/v1/devices":
            body = self.rfile.read(int(self.headers.get("Content-Length", 0)))
            with open(PAYLOAD, "wb") as f:
                f.write(body)
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(b'{"result":"created","device_id":1,"pending_change_id":null}')
        else:
            self.send_response(404)
            self.end_headers()

    def log_message(self, *a):
        pass


print("helper server on :18080", flush=True)
HTTPServer(("127.0.0.1", 18080), H).serve_forever()
