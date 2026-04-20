import json
import logging
import os
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
from typing import Any

LOG_LEVEL = os.environ.get("LOG_LEVEL", "INFO").upper()
PORT = int(os.environ.get("COLLECTOR_PORT", "8001"))
PROCESSOR_HOST = os.environ.get("PROCESSOR_HOST", "localhost")
PROCESSOR_PORT = int(os.environ.get("PROCESSOR_PORT", "8002"))

logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("event-collector")

events_store: list[dict[str, Any]] = []


class EventHandler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:
        if self.path == "/health":
            self._json_response(200, {"status": "healthy", "service": "event-collector", "timestamp": time.time()})
        elif self.path == "/events":
            self._json_response(200, {"events": events_store, "count": len(events_store)})
        else:
            self._json_response(404, {"error": "not found"})

    def do_POST(self) -> None:
        if self.path == "/events":
            try:
                length = int(self.headers.get("Content-Length", 0))
                body = self.rfile.read(length)
                event = json.loads(body)
                if "type" not in event or "data" not in event:
                    self._json_response(400, {"error": "event must have 'type' and 'data' fields"})
                    return
                event["received_at"] = time.time()
                event["id"] = len(events_store) + 1
                events_store.append(event)
                logger.info("Collected event id=%d type=%s", event["id"], event["type"])
                self._json_response(201, {"id": event["id"], "status": "collected"})
            except json.JSONDecodeError:
                self._json_response(400, {"error": "invalid JSON"})
            except Exception as e:
                logger.error("Failed to process event: %s", e)
                self._json_response(500, {"error": str(e)})
        else:
            self._json_response(404, {"error": "not found"})

    def _json_response(self, status: int, data: dict) -> None:
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())

    def log_message(self, format: str, *args: Any) -> None:
        logger.debug(format, *args)


def create_server(port: int | None = None) -> HTTPServer:
    return HTTPServer(("0.0.0.0", port or PORT), EventHandler)


def main() -> None:
    server = create_server()
    logger.info("Event Collector starting on port %d", PORT)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info("Shutting down")
        server.server_close()


if __name__ == "__main__":
    main()
