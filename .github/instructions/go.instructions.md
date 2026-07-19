---
applyTo: "**/*.go"
---

When reviewing Go changes:

- Follow standard library and existing package patterns in `internal/` and `cmd/`.
- Prefer table-driven tests in `*_test.go` files colocated with the code under test.
- Integration tests belong in `test/integration/` with the `integration` build tag.
- Avoid expanding public API surface without a clear need; `testkit` is the supported external test helper.
