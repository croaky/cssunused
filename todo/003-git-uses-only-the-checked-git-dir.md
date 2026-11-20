# 003: git reads its repository only from the .git entry of -root

Title: `security: start git only on the .git entry of -root`

Finding: `find_018ziUstuhpmggCYtafTxncq`.

## Need

The `sourceFiles` Go function starts `git ls-files` when `os.Lstat`
finds an entry named `.git` in `-root`. `Lstat` proves that the entry
exists, not that git accepts it. git then finds the repository itself.
When git rejects the `.git` entry, it reads `root` as a bare
repository. The change under check can supply `HEAD`, `objects/`,
`refs/`, and a `config` there. A `core.fsmonitor` key in that config
runs a program of the attacker's choice as the check user.

A probe against 0bf0335 ran the helper in two cases: an empty `.git`
directory, and a dangling `.git` symlink. A FIFO `.git` made git exit
with status 128, and no helper ran. The change cannot create the
`.git` entry itself, so the attack needs a rare condition on the
runner, such as a masked volume or a pruned worktree. The threat model
in `todo/README.md` does not accept code execution, whatever the
condition.

The README section on the scan and the doc comment of `sourceFiles`
say that a change cannot supply the config. That claim is false at
0bf0335.

## Change

1. `sourceFiles` reads the `.git` entry with `os.Lstat` and accepts
   two forms only: a directory, or a regular file. Any other form,
   such as a symlink, a FIFO, or a device, fails the run.
2. git starts with an explicit repository, so it does no discovery:
   `git -c safe.bareRepository=explicit --git-dir=<root>/.git
   --work-tree=<root> ls-files -z --cached --others
   --exclude-standard`. The command also sets
   `GIT_CEILING_DIRECTORIES` to the parent of `root`.
3. A directory `.git` that git rejects, such as an empty one, makes
   git fail. `sourceFiles` returns that error, as it does now for a
   bad gitfile.
4. Change the README section on the scan and the doc comment of
   `sourceFiles` in the same commit.

`safe.bareRepository` is read only from protected config, and `-c` is
protected. It needs git 2.38 or later.

## Tests

Put each test next to `TestRefsStartsNoGitOutsideACheckout`, and use
`plantRepo` for the bare-repository files.

- An empty `.git` directory: the run fails, and no marker file exists.
- A dangling `.git` symlink: the run fails, and no marker file exists.
- A worktree from `git worktree add`, with a `.git` file: the run lists
  the files that git lists. This test decides whether `--git-dir`
  accepts a gitfile. If it does not, resolve the gitfile in Go first,
  and pass the directory it names.
- `TestRefsReadsTheFilesGitLists` stays as it is.

## Cost

About ten lines and three tests. No new dependency and no new flag.
Speed does not change.

## Options

- A. The change above. Recommended.
- B. Remove git, and add a flag that names each build directory to
  skip. This removes every git config helper for good. Each app must
  then copy its ignore list into the flag. Take it if a third finding
  goes through git.

## Rollout

Tag a release after 004 lands, and bump each app that pins
cssunused in the same sitting. Then run
`claudesec resolve --reason fixed find_018ziUstuhpmggCYtafTxncq` with
a note that names the commit.
