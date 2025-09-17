import { NodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { PeriodicExportingMetricReader } from '@opentelemetry/sdk-metrics';
import { OTLPMetricExporter } from '@opentelemetry/exporter-metrics-otlp-grpc';

const endpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT || 'http://otel-collector:4317';

const traceExporter = new OTLPTraceExporter({ url: endpoint });
const metricReader = new PeriodicExportingMetricReader({
  exporter: new OTLPMetricExporter({ url: endpoint }),
  exportIntervalMillis: 10000,
});

const sdk = new NodeSDK({
  traceExporter,
  metricReader,
  instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();

const shutdown = () => {
  sdk.shutdown().catch((err) => console.error('error shutting down OpenTelemetry SDK', err));
};
process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);

