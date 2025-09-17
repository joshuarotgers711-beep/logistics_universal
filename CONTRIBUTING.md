# Contributing to Logistics Universal

Thanks for your interest in contributing! We welcome issues, docs improvements, and code contributions.

## Code of Conduct
By participating, you agree to uphold a respectful and inclusive community. Be kind, be constructive.

## Getting Started
- Fork the repo and create a feature branch from the latest default branch (e.g. `main`).
- Branch naming: `feat/<topic>`, `fix/<topic>`, `docs/<topic>`, `ops/<topic>`.
- For large changes, open a design issue first to discuss the approach.

## Development Setup
- Prereqs: Docker, Docker Compose; Node 20, Go 1.22, Python 3.11 (optional for local runs)
- Local dev guide: see `docs/LOCAL_DEV.md`.
- Observability runbook: `docs/runbooks/observability.md`.

## Testing
- Prefer small, focused unit tests; add integration tests where helpful.
- For end-to-end validation, use scripts under `scripts/` (e.g. `scripts/mtls_validation.sh`).

## Commit Messages
- Use clear, descriptive messages. Examples:
  - `feat(gateway): add mTLS client auth to outbound calls`
  - `fix(pricing): correct p95 latency panel query`
  - `docs: add setup instructions for local Jaeger`

## Pull Requests
- Keep PRs small and focused; include context in the description.
- Link related issues and describe testing steps.
- Ensure CI is green and linting passes.

## Security
- Do not commit secrets. Use the `*_FILE` pattern for secrets support.
- Report vulnerabilities privately—open a security advisory or contact maintainers.

## License
By contributing, you agree your contributions will be licensed under the MIT License (see `LICENSE`).

