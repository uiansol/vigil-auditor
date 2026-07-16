"""Entrypoint: HTTP /healthz + gRPC Auditor stub on concurrent ports."""

from __future__ import annotations

import logging
import os
import threading
from concurrent import futures

import grpc
import uvicorn
from fastapi import FastAPI

from app.pb import auditor_pb2_grpc
from app.servicer import AuditorServicer

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("vigil-ai")

HTTP_ADDR = os.getenv("AI_HTTP_ADDR", "0.0.0.0:8081")
GRPC_ADDR = os.getenv("AI_GRPC_ADDR", "0.0.0.0:50051")

api = FastAPI(title="Vigil AI Auditor", version="0.0.1")


@api.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


def serve_grpc() -> None:
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    auditor_pb2_grpc.add_AuditorServicer_to_server(AuditorServicer(), server)
    server.add_insecure_port(GRPC_ADDR)
    server.start()
    logger.info("gRPC auditor listening on %s", GRPC_ADDR)
    server.wait_for_termination()


def main() -> None:
    grpc_thread = threading.Thread(target=serve_grpc, name="grpc-server", daemon=True)
    grpc_thread.start()

    host, _, port_s = HTTP_ADDR.partition(":")
    if not port_s:
        host, port_s = "0.0.0.0", host
    port = int(port_s)
    logger.info("HTTP health listening on %s:%s", host, port)
    uvicorn.run(api, host=host, port=port, log_level="info")


if __name__ == "__main__":
    main()
