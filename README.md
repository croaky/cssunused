# cssunused

A check for a plain CSS stylesheet in a server-rendered Go app. It
reports class selectors that nothing renders, and rules that break a
layered CSS standard. CI runs it, so the build fails on the change that
orphans a selector or breaks the standard.

The standard is
[web / css](https://dancroak.com/web/css): six cascade layers in one
order, kebab-case component classes with no `__`, one token scale, and
breakpoints written as ranges.

## Install

Pin it as a tool of the app's module, so a check runs the version the
diff names:

```sh
go get -tool github.com/croaky/cssunused@latest
go tool cssunused -sheet ui/css/app.css
```

## Flags

- `-sheet path`: an app stylesheet to check. Repeat it for each sheet.
- `-root dir`: the directory of the app's templates, scripts, and Go.
  The default is the current directory.
- `-module path`: a required Go module whose sheet the app serves, such
  as `github.com/croaky/chart`. Repeat it for each module. The command
  reads the module's `.css` files for the classes it defines, and its
  views, scripts, and Go for the classes it writes. It finds the module
  with `go list -m`, so the app's `go.mod` must require it. The lint
  does not read a module's sheet, because the module keeps the rules of
  its own repo.
- `-builder name`: a script function whose arguments hold a class, such
  as `div("row")`. Repeat it for each function.

The command prints one line for each issue and exits 1 when it finds
one.

## How it decides

Defined: the class names in the resolved selectors of each sheet.
Nesting resolves, so `.card-slim` under `.card` counts as both, and
`.drawer &` under `.card` counts as `.drawer` too.

Referenced: a name read out of a position that puts a class on an
element, not a word read out of a file.

- `.hml`: the selector chain at the head of a line (`%header.card.flex`,
  `.mb-1.flex-2`), and the literal part of a `class:` attribute, around
  any `#{...}` interpolation. `klass:` counts too, since that is the
  name a partial's class local wears. A `:javascript` block is read
  with the script rules.
- `.html`: `class=` attributes.
- `.ts`, `.tsx`, and `.js`: a `className`, the arguments of a
  `classList` call, the `.class` selectors inside a `querySelector`,
  `closest`, or `matches` string, and the arguments of each `-builder`
  function.
- `.go`: parsed, not scanned. A literal counts in two positions: the
  body of a function named for a class (`statusClass`), and the value
  of a map key, struct field, or variable named for one
  (`"status_dot_class": ...`, `mainClass := "no-nav"`).

In a git checkout, the scan reads the files git lists: the tracked
files, and the untracked ones that `.gitignore` does not cover. A stale
bundle in an ignored build directory names the classes of an old
version of the app, so it vouches for nothing. Outside a checkout, such
as a module in the module cache, the scan reads every file. Either way
it skips `.git`, `node_modules`, `testdata`, and `vendor`.

An empty result from the parse of a sheet is an error, not a pass: it
means the sheet moved or the parse broke. A Go file the parser cannot
read is an error for the same reason.

## No allowlist

A class assembled at run time is invisible to this and gets reported.
That is the point, and the fix is to spell the name in source rather
than to excuse it: a status tag is a lookup that returns whole literals,
not `"tag tag-" +` the status.

A scan of words instead of positions can never report `.example`,
`.hint`, or `.tag`, because each is an ordinary English word. And an
allowlist that covers a family of names in one line is how a whole
utility scale gets deleted as unused while a `Sprintf` goes on naming
it.

## Deleting a hit

The tool reports, and a person deletes. Read the rule before removing
it, and look at the page in a browser. A hit is one of two things: a
selector nothing renders, which goes, or a name something still builds
at run time, which gets spelled out first and then stops being a hit.

## The lint

A class in any position listed above has no `__`, even when no sheet
defines it. A branch cut before a rename can still write an old element
name, and no other check reads a class the sheet does not define.

Each class follows the naming rule of its layer:

- `components` and `print`: a kebab-case name with an optional
  `--modifier` (`label--bold`), and no `__`. An element is a nested
  selector under its block (`.card { .title {} }`).
- `utilities`: a short name with no `__` and no `--` (`mt-1`).
- Every other layer: no class.

In `components`, each selector of a top-level rule starts with a class
(`.card`, `:is(.a, .b) .c`). A rule for a bare tag or an attribute goes
in `base`. A nested rule is not checked, because it starts with its
parent.

A selector in any layer fails when it names an ID (`#main_tabs`). An
ID outranks every class rule in its layer. A template keeps an ID that
a script reads, and the sheet styles a class on the same element. A `#`
inside an attribute value (`[href="#top"]`) names no ID.

A rule outside every layer fails, with a class or without. It beats
every layered rule, whatever its specificity, so one added below the
layers would override the utilities.

A media query that tests `min-width` or `max-width` fails. Write the
width as a range, `(width < 70em)`. `max-width: 70em` and
`min-width: 70em` both match at 70em, so a screen that wide gets both
layouts.

`!important` fails outside the `base` layer and the `.hidden` utility.
A later layer beats an earlier one without it. In `base` it beats a
style that a browser extension writes inline, and `.hidden` must beat
every display utility.

## GitHub repo is a mirror

Development happens on [sockeye](https://sockeye.sh), a self-hosted
review and CI server, which holds in progress branches. GitHub receives
`main` and the tags so `go get` works.

## License

MIT
