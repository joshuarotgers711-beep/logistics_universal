# RabbitMQ DLQ Runbook

Symptoms:
- DLQ message count stays > 0 for >5m
- DLQ has zero consumers while backlog exists

Quick checks:
1) Open RabbitMQ management UI (http://localhost:15672) and inspect DLQ queues
2) Verify consumer services are running and connected
3) Check application logs for nack/reject errors and routing keys

Possible causes:
- Consumer crashes or connection issues
- Message schema changes causing rejections
- TTL/dead-letter policies misconfigured

Actions:
- Restart/scale consumer service(s)
- Inspect a few DLQ messages to identify poison patterns
- Fix producer/consumer schema mismatch and requeue dead letters if safe

Metrics to watch:
- rabbitmq_queue_messages_ready{queue=~".*.dlq"}
- rabbitmq_queue_consumers{queue=~".*.dlq"}
- Consumer lag rates

