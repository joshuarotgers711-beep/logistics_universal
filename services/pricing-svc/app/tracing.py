import os
from opentelemetry import trace, metrics
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter


def init_tracing(app) -> None:
    # Idempotent init
    if getattr(app.state, "otel_initialized", False):
        return

    endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel-collector:4317")
    # grpc exporter expects host:port and separate insecure flag
    ep = endpoint.replace("http://", "").replace("https://", "")
    service_name = os.getenv("OTEL_SERVICE_NAME", "pricing-svc")

    resource = Resource.create({
        "service.name": service_name,
    })

    # Traces
    tprovider = TracerProvider(resource=resource)
    texporter = OTLPSpanExporter(endpoint=ep, insecure=True, timeout=5)
    tprovider.add_span_processor(BatchSpanProcessor(texporter))
    trace.set_tracer_provider(tprovider)

    # Metrics
    mexporter = OTLPMetricExporter(endpoint=ep, insecure=True, timeout=5)
    mreader = PeriodicExportingMetricReader(mexporter, export_interval_millis=10000)
    mprovider = MeterProvider(resource=resource, metric_readers=[mreader])
    metrics.set_meter_provider(mprovider)

    # Auto-instrument FastAPI and outgoing requests
    FastAPIInstrumentor.instrument_app(app)
    RequestsInstrumentor().instrument()

    app.state.otel_initialized = True

