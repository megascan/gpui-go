# Fixer brief, round 1

**Repo:** /home/strange/Projects/gpui-go-wt/sec, branch feat/security-sweep. Read docs/security/LEDGER.md: Threat model, Test profile, your SEC rows, and your cluster in "Fix clusters". Also read AGENT.md.

## Rules
- Edit only your cluster's files plus new test files in their packages. Other fixers are editing other files of the same tree at the same time. Transient compile errors in files you don't own are not yours: wait and retry. Never touch them.
- Do not edit `docs/port/**`, `TECH_DEBT.md`, LEDGER.md or the harness. Report what they need instead.
- No code comments. No new dependencies. Go idioms. Keep public API changes minimal, and report every one.
- For each SEC id you fix, add a regression test `TestSecurity_SEC<NNN>_<slug>` in the owning package (normal `_test.go`, no build tag) that failed before your change. Memory bombs: assert a clean error from the guard, not an OOM.
- Pick the conventional limit (example: 512 MiB decoded image budget = existing webpMaxCanvasBytes). Make limits package-level constants.

## Verify
1. `go test` and `go vet` on your packages, `gofmt -l` on your files.
2. Your harness scripts (`internal/securityharness/<sphere>/*.sh` or `run.sh`) must now PASS for your SEC ids.
3. `go run ./internal/cmd/paritycheck -lint`

Run all of this at `nice -n 19`, backgrounded when slow. Wait for your own jobs inside the turn.

## Commits
Do not commit, stash, checkout or reset.

## Don't
Sub-delegate.

## Final report
For each SEC id: fixed, not fixed, or partial; files; test name; harness result; any public API change; and a one-line proposed deviation text with its D-SEC id. End your turn only with this report.
