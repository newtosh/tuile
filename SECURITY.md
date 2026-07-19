# Security policy

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

### Preferred: private vulnerability reporting

Use [GitHub private vulnerability reporting](https://github.com/newtosh/tuile/security/advisories/new) on this repository. Reports stay confidential until we publish an advisory after a fix.

### Alternative

Open a draft [security advisory](https://github.com/newtosh/tuile/security/advisories) yourself if you are a maintainer triaging an external report.

## Scope

**In scope**

- `tuile serve` and the `/v1` HTTP + WebSocket API
- Session tokens, bootstrap secrets, and origin allowlist behavior
- Browser viewer (`/view`) XSS, CSWSH, or auth bypass
- `tuile-mcp` and `testkit` when they mishandle credentials or session access

**Out of scope**

- Vulnerabilities inside third-party agent CLIs (Claude Code, Codex, Cursor, etc.)
- Attacks that require running Tuile as root with `--allow-root`
- Deployments that intentionally bind to non-loopback addresses without TLS or a reverse proxy
- Social engineering of maintainers or users

## What to include

- Description and impact
- Steps to reproduce
- Tuile version (`tuile version`) or git tag
- Whether the server was bound to loopback (default) or exposed on a network

## Response expectations

We will acknowledge reports in a reasonable timeframe, ask for clarification when needed, and work toward a fix and coordinated disclosure. We do not offer a bug bounty at this time.

## Supported versions

| Version | Supported |
|---------|-----------|
| latest release | yes |
| `main` | yes, fixes land here first |
| older tags | best effort |
