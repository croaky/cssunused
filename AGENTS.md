# Agents guide

cssunused checks a plain CSS stylesheet in a server-rendered Go app: it
reports classes that nothing renders, and rules that break the layered
standard of `web/css` on dancroak.com. See `README.md` for the rules
and the flags.

## Writing

Write every word in ASD-STE100 Simplified Technical English (STE):
Markdown docs, code comments, commit messages, change descriptions,
and replies in an agent conversation.
See <https://en.wikipedia.org/wiki/Simplified_Technical_English>.

STE is a controlled English for technical writing: one meaning per
word, one idea per sentence, and the actor named. It is not a house
style. It exists so every reader reads a sentence the same way. That
includes a tired reader, a reader in a second language, and an agent
that matches on words.

- One idea per sentence. Keep an instruction to 20 words and a
  description to 25.
- Active voice, present tense, and the actor named: say what acts,
  rather than writing "the tick is placed".
- One word, one meaning. Keep a term the same everywhere rather than
  varying it for tone.
- Use the simple verb, not a noun made from it: "run the formatter",
  not "perform execution of the formatter".
- Cut what carries nothing: "simply", "just", "note that", "in order
  to".
- Put a warning or a limit before the step it applies to.
- STE limits a sentence, not a text. Keep each sentence short, but
  keep the sentence that defines a term or connects a cause to its
  effect.
- STE permits technical names. Name the rule, the flag, or the
  function, not a vague noun such as "the snapshot".

Apply it to prose, not to code: an identifier, a command, and a quoted
error message stay as they are.

This repo is public. Name no private app, company, or person in code,
docs, tests, or commits. Write an example with a neutral name.

## Architecture

One `main` package at the repo root, so an app pins it with
`go get -tool github.com/croaky/cssunused` and runs `go tool cssunused`.

- `main.go`: the flags, and `run`, which reads each sheet and each
  module and returns the report.
- `sheet.go`: `ParseSheet`, which reads each rule's own selector and
  records what the lint reads: the classes of each layer, rules outside
  a layer, IDs, width media queries, and `!important`.
- `refs.go`: `Refs`, which reads the class positions of templates,
  HTML, scripts, and Go.
- `lint.go`: `Lint`, the rules of the standard.

A rule of the standard lives in three places that change together: the
code, the doc comment of `Lint`, and the README's "The lint" section.
A change to one changes the other two in the same commit.

The tool keeps no allowlist and reads no app by name. A need of one app
is a flag, such as `-builder`, with a test.

## Checks

Pin every tool a check runs. `tools.mod` and `tools.sum` hold the
version and checksum of each tool. Run a tool as
`go tool -modfile=tools.mod <name>`. Add or update a tool with
`go get -modfile=tools.mod -tool <pkg>@<version>`. Do not run
`go run pkg@latest` or `go install pkg@latest` in a `Checkfile`, a
script, or CI. A moving version makes the same commit pass one week
and fail the next, and it runs new tool code nobody reviewed. A pinned
tool changes in a change, where the diff shows the update and CI runs
it.

Keep the tools out of `go.mod`. A `tool` line there puts the tool's
requirements in the module graph of every module that requires this
one. `tools.mod` names this module, but only
`go tool -modfile=tools.mod` reads it. The tool then reads `go.mod`,
so `govulncheck` scans the versions this module ships. Do not run
`go mod tidy -modfile=tools.mod`, because it copies the requirements
of `go.mod` into `tools.mod`.

Pin the tool, not its data. `govulncheck` reads the live vulnerability
database on every run, so the weekly scan still finds advisories
published after the commit. Never vendor or cache that data.

The root `Checkfile` is the list, and CI runs it on every push. Read it
there rather than here, and run it before committing. Two of its
entries check rather than write, since a CI job that rewrote source
would have nowhere to put it. Run the writing form first:

```sh
go tool -modfile=tools.mod goimports -local "$(go list -m)" -w .
dprint fmt
```

The command imports only the standard library. Tests also import
`github.com/croaky/is`. Taking another dependency is a design decision,
not a step.

## Tests

Red/green TDD. A test of a rule asserts the exact line the rule prints.
A test of `Refs` writes the files to `t.TempDir()` and asserts which
names count and which do not, because a name that counts by mistake
keeps a dead selector alive.

Assertions come from `github.com/croaky/is`. Pick the helper that
names the check, `Eq` and `NotEq` by default, `True` only for a
predicate with no want to name.

## Changes

Work happens on a sockeye change. `soc checkout` allocates one and
prints a worktree; `soc edit` sets its title and description. Do the
edit before the code, not after. A change with neither is a blank row on
the dashboard and a blank `soc show`, so nobody looking at either can
tell what it is or whether it overlaps what they are about to start. A
rough sentence beats an empty one, and the description gets rewritten
before the merge anyway.

Push with `soc push --wait`, which waits for the checks, rather than
sleeping and then reading. `--wait` follows the commit in the worktree
it runs in, exits nonzero when a check failed, and gives up after ten
minutes (`--timeout`). If `main` moved ahead, run `soc sync` to rebase
the change, then `soc push --force`. A plain push after a sync stops
and says so.

## Commits

- Subject: `scope: imperative summary`, at most 72 characters, lowercase
  except proper nouns, no trailing period.
- The scope is what the change acts on: `lint:` for a rule, `sheet:`
  for the parse of a sheet, `refs:` for the read of templates, scripts,
  and Go, and `cli:` for the flags and the report. Not `cssunused:`,
  because every commit here is cssunused.
- Each repo also uses these scopes: `security:` for a fix to a security
  finding, `deps:` for a dependency or the Go version, `doc:` for the
  docs, `todo:` for a plan, and `ci:` for the Checkfile and its checks.
- A change that acts on more than one area takes the scope whose
  behavior changed. Write two scopes, with a comma between them, only
  when each one changed: `lint, sheet:`.
- The `merge:` line in the root `Checkfile` checks the change title at
  merge. It caps the text after the scope at 64 characters.
- Include _why_, not just _what_. Hard-wrap the body at 72.
- Write the subject for a teammate who has not read the code. Name the
  rule, the flag, or the file type, not a term the change makes up.
- Write the body as a change description (below). The first push of a
  branch with one commit copies the commit message to the change.
- Sign your work with a `Co-Authored-By` trailer at the end of the
  message, after a blank line. A trailer on the line after the subject
  becomes part of the subject.

## Change descriptions

sockeye squashes a change into one commit whose message is the
change's title and description (`soc edit`). Write the description as
that commit message.

Write for an engineer reading it a year from now. They know Go and
Git. They did not see your conversation, and they do not have the diff
open.

Put the most important fact first. A reader who stops after any
paragraph has the most important part so far. Use this order, and
leave out a part that does not apply:

1. The need. Name who uses what, what went wrong or was missing, and
   what the change does about it.
2. What a user sees now. Name the rule, the flag, or the report line.
   Say what stops happening and what a user does differently. If no
   user sees the change, say what it protects: a test, a release, or a
   cost.
3. How it works. Name each part at its first mention, with its
   kind: "the `ParseSheet` Go function", "the `-module` flag".
   Give the cause of a bug or the rule a feature applies, and the
   numbers that size the effect.
4. Next steps: a command to run, a follow-up plan by its path, or a
   known gap.

Most changes fit in three to five short paragraphs. To keep a
description short and clear:

- Define a project term at its first use, or use a plainer word.
- Leave out file lists, line counts, and code sizes. The diff shows
  them.
- Leave out what a reader of `main` cannot use: options you did not
  take, each edge case the tests cover, and "tests pass". Put a design
  argument in a code comment or a plan, and give its path.
- Use plain paragraphs. Git strips a `#` line as a comment, so a
  Markdown header disappears.
- Hard-wrap at 72 columns, except a trailer. Don't backslash-escape
  backticks. Use a quoted heredoc such as `<<'EOF'` with `soc edit`.
- Keep `Co-Authored-By` on the commits, not in the description. The
  merge collects the `Co-Authored-By` trailers from the commits it
  squashes.

## Trailers

End the change description with one block of trailers, with no blank
line between them. Keep each trailer on one line, even past 72
columns.

- `Requested-by: name, channel, thread or subject, date` records who
  asked for the work, when somebody other than the author asked.
- `Reported-by: Claude Security at claude.ai/security` credits a
  finding from the Claude Security scanner.
- `Fixes:` names each Claude Security finding the change fixes
  (`Fixes: find_01... and find_02...`), or each production error group
  (`Fixes: ERR-1F`). Do not write `Fixes: CWE-...`. A CWE names a
  category, not a finding.
- Do not type `Co-Authored-By`, `Reviewed-by`, or `Signed-off-by` in
  the description. The merge derives them from the commits and the
  comments.

The squash merge writes the description to `main`, not the commit
messages. The first push of a branch with one commit copies its
trailers to the description, except the ones the merge derives. A
later `soc edit` writes only what it reads, so keep the trailers in
it:

    git --no-pager log -1 --format=%B | sed '/^Co-Authored-By:/d' | soc edit

Before the merge, read the description in `soc show` and confirm that
it ends with the trailers.

## Releases

sockeye is origin and holds no tags. `scripts/tag vX.Y.Z` publishes one
annotated tag to the GitHub mirror, which is what a `go get` resolves.

No user should discover a break from a tag. `scripts/tag` refuses a
dirty tree and a `main` behind the server's, but it cannot know whether
the code is any good to the apps that will fetch it. So before tagging,
run each user's check against the working tree:

```sh
go run /path/to/cssunused -sheet <the app's sheet> ...
```

Bump each user in the same sitting. A new rule fails each app that
breaks it on the next push after the bump. So the change that bumps an
app also fixes what the new rule finds there, and the change
description of the rule names it as a breaking change.
