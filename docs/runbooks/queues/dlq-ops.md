# DLQ Operations (Replay & Quarantine) Runbook

Tooling: tools/dlqctl (container: lmp-dlqctl)

Common commands:
- Analyze one peek message per DLQ: docker compose run --rm --profile tools dlqctl analyze
- Peek N messages: docker compose run --rm --profile tools dlqctl peek --dlq shipments.dlq --limit 5
- Dry-run replay (no mutations): docker compose run --rm --profile tools dlqctl replay --dry-run
- Replay with quarantine threshold: docker compose run --rm --profile tools dlqctl replay --poison-threshold 3

Poison detection:
- Uses RabbitMQ x-death header count; messages with count >= threshold are sent to <base>.poison instead of original queue

Schema versioning:
- Publishers should set header x-schema-version (Compose SCHEMA_VERSION)
- Mismatches often indicate producer/consumer version drift; roll forward or back to compatible versions

Metrics:
- dlqctl_replayed_total{queue}
- dlqctl_quarantined_total{queue,reason}
- dlqctl_analyzed_total{queue}

Cautions:
- Always start with --dry-run in production
- Verify consumers are healthy before replay

