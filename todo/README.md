# Todo

Each file here is one sockeye change. The number gives the order in
which the changes land. The commit that finishes a change deletes its
file. The next file takes a number past the highest one in use.

- `001-git-only-in-the-app-checkout.md`: `refs:` starts git only in
  the app's own checkout.
- `002-sheet-reads-each-rule-own-selector.md`: `sheet:` stops the
  expansion of nested selectors.

The two changes touch different files, so either one can land first.

Every change runs the `Checkfile` list before it opens.

## Security findings

Claude Security scans this repo. `claudesec show <id>` prints the
attack path and the fix criteria of a finding, so this file keeps no
copy. A plan names each finding that its change acts on.

A change that fixes a finding removes the ID from this index. After
the change lands, the owner runs `claudesec resolve`. A finding that
this threat model accepts gets `--reason accepted_risk` and a note
that names the attacker, the consequence, and the boundary.

### Open findings

- `find_01NEnV2pjxcSJyY7mxGf7tFD`: `git ls-files` runs in a module
  directory and obeys its repository config. Fix:
  `001-git-only-in-the-app-checkout.md`.
- `find_01VSfRs8EY66pzqUajFevuHP`: `resolve` expands nested selectors
  without a limit. Fix by removal:
  `002-sheet-reads-each-rule-own-selector.md`.
- `find_0133THTx3czBjpxFF46HFBqN`: each `!important` declaration
  copies the whole selector list. Plan 002 removes the expanded list.
  The rest of the risk is accepted below.
- `find_01YM9WKjrBLouA7pZno1xEfQ`: `startsWithClass` takes quadratic
  time on a nested `:is(` selector. Accepted below.

## Threat model

This section is a proposal. The owners confirm it before a finding is
resolved against it.

### What cssunused is

cssunused is a command-line check. It runs in the CI job of an app,
and on the machine of a developer of that app. It has no server, no
network listener, and no credential. It reads files and prints lines.
The app can be on the public internet, but cssunused is not part of
the app at run time.

### Adversaries, most likely first

1. The author of a change to an app. The author controls the content
   of each file in the change, the app's `go.mod` included. The same
   CI job runs `go test` on that change, so the author already runs
   code there as the check user.
2. The author of a Go module that the app requires. The module's code
   is compiled into the app and runs in the app's tests and in
   production.
3. A developer who fetches a branch and runs only cssunused on it,
   with no `go test`. This is the one case where cssunused is the
   only program that reads the hostile files.

### Accepted risks

- A hostile file makes the check slow, uses much memory, or fills the
  CI log. The attacker is adversary 1 or 2. The consequence is a
  failed or timed-out run of the check, and the check never reports a
  false pass. The boundary is the CI job of the attacker's own change.
  The attacker can get the same result with a loop in a test, so a
  limit in cssunused protects nothing. A limit adds code to each
  parser path. This covers `find_01YM9WKjrBLouA7pZno1xEfQ` and the
  part of `find_0133THTx3czBjpxFF46HFBqN` that plan 002 leaves.
- Code execution through a file is not accepted. Adversary 3 runs no
  test, so cssunused must not start a program that file content
  chooses. Plan 001 closes the one path the scan found.
