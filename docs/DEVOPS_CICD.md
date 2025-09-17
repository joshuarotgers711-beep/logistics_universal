# DevOps, Environments, and CI/CD

## Environments
- Dev: fast iteration, feature flags always on, seeded data
- Staging: production-like, performance tests, security scans, UAT
- Prod: blue/green or canary, strict change controls, on-call

## Branching & Releases
- Trunk-based development; short-lived feature branches; PR with checks
- Semantic versioning per service; automated changelogs

## CI Pipeline (per service)
1) Lint/type check
2) Unit tests (coverage gates)
3) Build container image (SBOM, signing)
4) SCA + SAST scans
5) Contract tests (provider/consumer)
6) Integration tests with ephemeral env (Kind + Testcontainers)
7) Push to registry on main; Helm chart bump

## CD Pipeline
- GitOps (Argo CD) manages desired state per env
- Progressive delivery (canary) with auto rollback on SLO breach

## Kubernetes
- Namespaces per env; HPA; PDB; resource requests/limits
- Secrets via Vault CSI; ConfigMaps for env config
- Ingress controller with WAF; mTLS possible between services

## Observability
- OpenTelemetry SDK; Prometheus/Grafana; Loki for logs
- SLOs defined; alerts routed via PagerDuty/ops channel

## Backups & DR
- Automated Postgres backups; PITR; restore drills quarterly
- Object storage versioning for labels/media

## Runbooks & Docs
- Standard runbook template: overview, KPIs, dashboards, alerts, ops tasks, failure modes
- On-call rotation and escalation policy

