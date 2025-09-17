# Security Policy

We take the security of Logistics Universal seriously. If you believe you have found a security vulnerability, please report it responsibly.

## Supported Versions

We aim to patch the default branch and the latest tagged release (if applicable).

## Reporting a Vulnerability

- Do not create a public GitHub issue for security reports.
- Preferred: open a private GitHub Security Advisory in this repository (Security tab → “Report a vulnerability”).
- Alternatively, email the maintainers with details (proof-of-concept, impact, affected components). Avoid including secrets or sensitive data.

We will acknowledge receipt within 3 business days and aim to provide an initial assessment within 7 business days.

## Contact

- GitHub Security Advisories: use the repository Security tab
- Maintainer ping: mention @joshuarotgers in the advisory if escalation is needed

## Security Best Practices in This Repo

- No plaintext secrets in source control; use `*_FILE` env pattern and Docker secrets.
- mTLS recommended for service-to-service traffic in production.
- CI runs linters and tests on all PRs; please keep changes small and focused.

## Disclosure Policy

We follow responsible disclosure. We will coordinate a fix and publish advisories with CVE (as appropriate).
