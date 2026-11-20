# Todo

Each file here is one sockeye change. The number gives the order in
which the changes land. The commit that finishes a change deletes its
file. The next file takes a number past the highest one ever used, so
a number in an old commit names one plan only.

- `004-reads-go-through-os-root.md`: `security:` every read stays in
  `-root` or in the module cache.
- `005-layer-set-once-per-block.md`: `security:` `ParseSheet` finds
  the class set of a layer once for each block.
- `006-threat-model-in-readme.md`: `doc:` the threat model moves to
  `README.md`.

005 changes only `sheet.go`, and 006 changes only docs, so each can
land in parallel with the others.

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

Each finding below was scanned at 0bf0335. A probe against a build of
that commit, with git 2.56.1 on APFS, proved each path, except where a
line says otherwise.

- `find_019yhBgMRaUinUardifzkzTM` (low): a `-root` that is a symlink
  moves the git listing to another repository. The probe printed a
  class of that repository. The second path, a case-colliding
  directory and symlink, did not reproduce: the clone wrote a real
  directory. An older git on a runner can still decide it. This is a
  sibling of the fix in 176f488. Fix: `004-reads-go-through-os-root.md`.
- `find_0188ZgZdXn7g7dbYmJZTXwZ1` (low): a `replace` directive in the
  app's `go.mod` points `-module` at any directory, and `readModule`
  reads it. The probe printed a class of an outside sheet. Fix:
  `004-reads-go-through-os-root.md`.
- `find_01MtCdmmcqwdTQh9jqaVFr9W` (low): `readSheet` checks only the
  last component of the `-sheet` path. A symlinked parent directory
  makes it read an outside sheet. This is a sibling of the fix in
  176f488. Fix: `004-reads-go-through-os-root.md`.
- `find_015Z9z7aWRTgcTUhL8BHih1r` (low): `ParseSheet` hashes the whole
  layer name for each class. The probe took 0.17 s for 200 KB, 0.58 s
  for 400 KB, and 1.59 s for 800 KB. Fix:
  `005-layer-set-once-per-block.md`.

## Threat model

The resolution notes of the earlier findings cite this model. Commit
6e93b0d deleted it with the old index, so plan 006 moves it to
`README.md`, where a fix cannot delete it with its plan.

### What cssunused is

cssunused is a command-line check. It runs in the CI job of an app,
and on the machine of a developer of that app. It has no server, no
network listener, and no credential. It reads files and prints lines.
The app can be on the public internet, but cssunused is not part of
the app at run time.

### Adversaries, most likely first

1. The author of a change to an app. The author controls the content
   of each file in the change, the app's `go.mod` included. Each app
   that pins cssunused also runs `go test` in its `Checkfile`, so the
   author already runs code in CI as the check user. Whether each
   `Checkfile` entry shares one user and one file system on the
   runner is not yet confirmed.
2. The author of a Go module that the app requires. The module's code
   is compiled into the app and runs in the app's tests and in
   production.
3. A developer who fetches a branch and runs only cssunused on it,
   with no `go test`. This is the one case where cssunused is the
   only program that reads the hostile files. The report goes to the
   developer's terminal, not to the attacker.

### Accepted risks

- A hostile file makes the check slow, uses much memory, or fills the
  CI log. The attacker is adversary 1 or 2. The consequence is a
  failed or timed-out run of the check, and the check never reports a
  false pass. The boundary is the CI job of the attacker's own change.
  The attacker can get the same result with a loop in a test. A fix
  that costs a few lines at one site still lands, as 005 does.
- Code execution through a file is not accepted. Adversary 3 runs no
  test, so cssunused must not start a program that file content
  chooses.
- A read outside the checkout is not accepted, although adversary 1
  can read the same files from a test in CI. The fix of
  `find_01TeJCkoubKMyntsXL5BS88H` set this rule, and 004 keeps it.
