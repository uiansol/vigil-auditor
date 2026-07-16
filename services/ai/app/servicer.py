"""Stub Auditor gRPC servicer for Slice 0 (no real audit logic)."""

from __future__ import annotations

import grpc

from app.pb import auditor_pb2, auditor_pb2_grpc


class AuditorServicer(auditor_pb2_grpc.AuditorServicer):
    """Placeholder implementation — returns UNIMPLEMENTED until later slices."""

    def AuditStatement(self, request, context):  # noqa: N802 — gRPC method name
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details("AuditStatement not implemented in Slice 0")
        return
        yield  # pragma: no cover — makes this a generator for unary-stream typing
