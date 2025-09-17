# Database Audits (pg_stat_statements)

Enable:
- docker-compose starts Postgres with shared_preload_libraries=pg_stat_statements
- Init script creates extension on first boot

Collect:
- docker compose exec -T postgres psql -U postgres -d lmp -f scripts/db_audit/collect.sql > .artifacts/db/pg_stat_collect.txt
- EXPLAIN example:
  - scripts/db_audit/explain.sh "SELECT * FROM shipments WHERE tenant_id='demo' AND created_at > now()-interval '1 day'"

Interpretation tips:
- Look for high total_time / mean_time, large rows vs actual, sort/hash spills
- Add missing composite indexes, avoid sequential scans on hot paths, check JOIN order and filter selectivity

