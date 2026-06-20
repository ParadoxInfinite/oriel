<!-- Thanks for contributing to Oriel! See CONTRIBUTING.md before opening. -->

## What & why

<!-- What does this change, and what problem does it solve? Link any issue: Closes #123 -->

## How

<!-- Brief notes on the approach. If it touches an edition, say which (Studio / Classic)
     and confirm shared logic still lives behind the platform SDK. -->

## Checklist

- [ ] Scope is focused — unrelated fixes/refactors are split into their own PRs.
- [ ] Commits follow [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, …).
- [ ] `go build ./...`, `go vet ./...`, and `go test -race ./...` pass.
- [ ] `cd web && npm run build` succeeds (UI changes).
- [ ] Editions stay pure presentation — no new `../lib/*` imports outside the SDK (`web/src/platform/index.js`).
- [ ] Docs/CHANGELOG updated if behavior or flags changed.

## Screenshots

<!-- For UI changes, before/after in the affected edition(s). Delete if N/A. -->
