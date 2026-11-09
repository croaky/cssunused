# 001: start git only in the app's own checkout

Title: `refs: start git only in the app checkout`

Finding: `find_01NEnV2pjxcSJyY7mxGf7tFD`.

## Need

`sourceFiles` runs `git ls-files` in each directory that `Refs`
reads. That includes a `-module` directory and a `-root` with no
`.git` entry. git then searches for a repository from that directory.
A directory that holds `HEAD`, `objects/`, `refs/`, and a `config` is
a bare repository to git. A change to the app can add those files and
point a `go.mod` `replace` at them. A module author can ship them in
the module zip. git reads that `config`, and a key such as
`core.fsmonitor` starts a program of the attacker's choice.

In CI the change author already runs code through `go test`. The
real gap is adversary 3 in `README.md`: a developer who runs
only cssunused on a fetched branch.

`sourceFiles` also discards the git error and walks the tree. The
report then depends on which listing ran, and no line says which one.

## Change

1. `sourceFiles` runs git only when `root/.git` exists, as a
   directory or as a worktree file. A change cannot add a `.git`
   entry, because git refuses to track that path. When `root/.git`
   exists, git finds it before it reads any file of the change.
2. When `root/.git` exists and git fails, `Refs` returns the error.
   The silent fallback to `WalkDir` goes away.
3. A module directory always uses the walk. `readModule` calls the
   walk directly, with no git. A copy in the module cache is not a
   checkout, and the doc comment of `sourceFiles` already says so.

## Tests

- A root with `HEAD`, `objects/`, `refs/`, and a `config` that sets
  `core.fsmonitor` to a command that writes a marker file. Assert no
  marker file, and assert the walk's file list.
- The same files in a module directory, read through `readModule`.
- `TestRefsReadsTheFilesGitLists` stays as it is.

## Cost

About ten lines and two tests. No new dependency and no new flag.
Speed does not change: a walk and `git ls-files` cost about the same.

## Options

- A. The change above. Recommended.
- B. Accept the risk. Zero code. Adversary 3 can run a program as the
  developer. The `-module` doc says the command reads a `replace`
  target, so the path is documented and easy to reach.
- C. Find the checkout in Go: walk up from `root` to the first `.git`
  entry, then run git with `--git-dir` and `--work-tree` set. This
  keeps the git listing for a `-root` inside a monorepo. It adds about
  ten more lines. Take it only when an app runs with such a `-root`.

## Known gap

A `replace` target that is a local checkout now uses the walk. A
stale build output in that checkout can then name a class and keep a
dead rule alive. The report misses a dead rule, but it never fails
by mistake.

## Docs

The README section on the scan says when git runs. Change it in the
same commit.
