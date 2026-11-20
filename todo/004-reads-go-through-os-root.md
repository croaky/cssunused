# 004: every read stays in -root or in the module cache

Title: `security: read files only through os.Root`

Findings: `find_019yhBgMRaUinUardifzkzTM`,
`find_01MtCdmmcqwdTQh9jqaVFr9W`, and `find_0188ZgZdXn7g7dbYmJZTXwZ1`.

Plan 003 landed first. It made `sourceFiles` start git with an
explicit `--git-dir` on the `.git` entry of an absolute `-root`. Keep
that call when `sourceFiles` moves to `os.Root`.

## Need

The fix in 176f488 stopped reads through a symlink. It checks each
path with `os.Lstat`, then opens it again by name with `os.ReadFile`.
`Lstat` checks only the last component of a path. The OS follows a
symlink in `-root` and in each middle component. Three findings go
around that check, and a probe against 0bf0335 proved each one:

- `readSheet`: `-sheet ui/css/app.css`, with `ui` a symlink, read an
  outside sheet and printed its class.
- `sourceFiles`: `-root web`, with `web` a symlink to a second
  repository, listed and read that repository.
- `moduleDir`: a `replace` directive in the app's `go.mod` named an
  outside directory, and `readModule` read it. Here the change chooses
  the read root itself, so a symlink check alone does not fix it.

The second path of `find_019yhBgMRaUinUardifzkzTM`, a case-colliding
directory and symlink, did not reproduce with git 2.56.1 on APFS. The
change below covers it whatever git does.

## Change

1. Open `-root` once with `os.OpenRoot`. A `-root` that is itself a
   symlink fails the run.
2. Read each `-sheet` file through that root. A `-sheet` path outside
   `-root` fails the run. The test and the read then act on one opened
   file, so the race between them goes away.
3. Read each path that git lists through that root. `os.Root` refuses
   a path whose symlink leaves the root.
4. `walkFiles` and `moduleClasses` walk with `fs.WalkDir` over
   `root.FS()`. They keep the rule that only a regular file counts.
5. `moduleDir` resolves the `Dir` that `go list -m` prints with
   `filepath.EvalSymlinks`. It accepts a directory inside the resolved
   `GOMODCACHE` or inside the resolved `-root`. Any other directory
   fails the run before any file there is listed. The error quotes the
   path with `%q`.
6. A new `-module-dir` flag lets an operator allow one outside
   directory for a local `replace` experiment. The operator names it
   on the command line, so the change under check cannot choose it.
7. Change the README sections on `-sheet`, `-module`, and the scan in
   the same commit.

## Tests

- A `-sheet` whose parent directory is a symlink to an outside
  directory with a same-named sheet: the run fails, and no class of
  that sheet appears.
- A `-root` that is a symlink to a second git repository: the run
  fails, and no file of that repository counts.
- A git index that lists `dir/file.ts` while `dir` on disk is a
  symlink to an outside directory: no class of the outside file counts.
- A `go.mod` whose `replace` names a directory outside `-root`: the run
  fails, and no class of that directory appears. With `-module-dir`
  set to that directory, the run reads it.
- A module in the module cache reads as before.

## Cost

- A `-sheet` must be inside `-root`. Each app that pins cssunused
  passes such a path today.
- A local `replace` experiment needs `-module-dir`. No app that pins
  cssunused has a local `replace` or a `go.work` today.
- `os.Root` opens each path component with `openat`, so a read costs a
  little more.
- This is a breaking change, so its description says so.

## Options

- A. The change above. Recommended.
- B. Add `filepath.EvalSymlinks` and a prefix check at each of the
  five read sites. Each site repeats the logic that 176f488 got partly
  wrong, and a new read site can forget it.
- C. Accept the risk, because adversary 1 can read the same files
  from `go test` in CI. The fix of `find_01TeJCkoubKMyntsXL5BS88H`
  already refused this argument, so B and C break that decision.

## Rollout

Tag a release, and bump each app that pins cssunused in the same
sitting. Run each app's check against the tree before the tag. Then
run `claudesec resolve --reason fixed` for each of the three IDs, with
a note that names the commit. The same tag ships the fix of 003, so
also resolve `find_018ziUstuhpmggCYtafTxncq` with a note that names
the 003 commit.
