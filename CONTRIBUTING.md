# Contributing to Tuile

Thanks for helping improve Tuile. This project is a PTY bridge for AI coding agents — contributions that make sessions easier to observe, automate, or test are especially welcome.

## Getting started

1. Fork and clone [github.com/newtosh/tuile](https://github.com/newtosh/tuile).
2. Install **Go 1.26+**.
3. Build and run locally:

```bash
make build
cp tuile.toml.example tuile.toml
./bin/tuile serve --force
```

See [README.md](README.md) for API examples, agent CLI setup, and viewer usage.

## Making changes

1. Create a branch from `main`.
2. Keep changes focused — one logical change per pull request when possible.
3. Match existing code style and naming in the files you touch.
4. Open a pull request with a short summary of **why** the change is needed and how you tested it.

Bug fixes and small improvements are great first contributions. For larger features, open an issue or draft PR early so we can align on approach.

## Running tests

```bash
make test              # fast unit tests (run before every PR)
make vet               # go vet
make race              # race detector (optional, slower)
make test-integration  # integration + browser tests (needs Chrome)
```

Integration tests live under `test/integration/` and use the `integration` build tag. Browser tests skip locally when Chrome is unavailable; CI installs Chromium via `.github/workflows/ci.yml`.

Downstream projects can reuse the public `testkit` package — see [docs/testing-with-tuile.md](docs/testing-with-tuile.md).

## Docs and screenshots

- Update [README.md](README.md) when behavior or setup changes.
- README screenshots should show real agent TUIs but **must not include personal names, emails, or account-specific strings**. Prefer login screens, in-progress agent output, or generic shell sessions.
- Put new images in `docs/images/`.

## What CI checks

Every push and pull request runs:

- `go mod tidy` check (module files must be up to date)
- `go vet ./...`
- `go build` for `cmd/tuile` and `cmd/tuile-mcp`
- `go test ./...` (unit tests)
- `go test -tags=integration ./test/integration/...` (integration + browser)

Please make sure both jobs pass before requesting review.

## Questions

Open a GitHub issue for bugs, feature ideas, or questions about how something should work.
