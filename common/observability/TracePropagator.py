# TracePropagator.py
"""
Distributed Tracing Propagator
------------------------------
Adds tracing headers for distributed monitoring across IAROS microservices.
Ensures end-to-end traceability for performance and failure analysis.
"""

import uuid

def add_trace_headers(headers: dict, trace_id: str = None) -> dict:
    if trace_id is None:
        trace_id = generate_trace_id()
    headers["X-Trace-ID"] = trace_id
    headers["X-B3-TraceId"] = trace_id
    headers["X-B3-SpanId"] = generate_span_id()
    return headers

def generate_trace_id() -> str:
    return uuid.uuid4().hex

def generate_span_id() -> str:
    return uuid.uuid4().hex
