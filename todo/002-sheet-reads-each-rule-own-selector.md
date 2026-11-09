# 002: read each rule's own selector, not the expanded list

Title: `sheet: report each nested rule by its own selector`

Findings: `find_01VSfRs8EY66pzqUajFevuHP` (fix by removal) and
`find_0133THTx3czBjpxFF46HFBqN` (most of it removed, the rest
accepted). `find_01YM9WKjrBLouA7pZno1xEfQ` touches `lint.go` beside
this change, and `README.md` accepts it.

## Need

`ParseSheet` calls `resolve` for each block. `resolve` builds every
combination of the block's selectors with the parent's list, and puts
the whole parent in place of each `&`. A sheet of about a hundred
bytes can use all the memory of the CI runner. Each `!important`
declaration then joins the whole expanded list into one more string.

The tool needs only three facts from a selector: its classes, whether
it names an ID, and its text for a report line. Each block's own head
holds all three. A class of a parent is already recorded at the
parent's block. A `&` adds no class that the parent did not add.

## Change

1. Delete `resolve`. Each block records the classes and the ID test
   of its own head.
2. The frame keeps the head of the nearest style rule and a flag that
   a style rule wraps it. The `ComponentsTopLevel` test reads that
   flag in place of `slices.Equal(top.sels, []string{""})`.
3. `IDSelectors` and `Important` record the rule's own head. A nested
   `.title` under `.card` now prints `.title`, not `.card .title`. A
   reader finds that text in the sheet with grep, which a resolved
   selector does not permit.
4. A `@keyframes` block still records no class.
5. The `.hidden` exception of the `!important` rule applies only to a
   top-level `.hidden` rule in `utilities`. Without this check, a
   nested `.hidden` under `.card` matches the own head and passes.
6. A `@layer` block nested in a style rule records only the classes
   of its own heads in that layer. Today the parent's classes go in
   that layer too. Add a test for this case before the change, and
   keep the old result if a sheet of an app depends on it.

## User-visible change

The `id:` and `important:` lines of a nested rule print the selector
as written. `TestParseSheetIDSelectors` and `TestParseSheetImportant`
change their wants. The doc comment of `Lint` and "The lint" section
of the README say how a nested rule is named. With steps 5 and 6, no
app fails where it passed before, and no app passes where it failed
before.

## Cost

The code gets smaller. `resolve` goes, and the per-block loop scans
one head in place of the whole inherited list. Parse time becomes
linear in the size of the sheet.

## What stays

Each `!important` line still copies its rule's head. A single rule
with a head of half the sheet and many short declarations still
makes lines of quadratic total size. A sheet of 100 KB gives about
200 MB. `README.md` accepts this. If the owners want a
bound, the options are:

- A. Accept. Zero code. Recommended, because the attacker can loop in
  a test of the same job.
- B. Cut the head in each report line at 80 bytes, with `...` at the
  end. About five lines. This also bounds the `no layer:`, `id:`, and
  `components:` lines.
- C. Print one `important:` line per rule, with all its declarations.
  This changes the line format again, and the size of the code stays
  the same.

The quadratic `:is(` scan in `startsWithClass` stays. A depth limit
of eight would make it linear in about three lines, with a lint
error past that depth. `README.md` accepts it in place of
that limit.

## Release

The report lines change, so the change description names the new
format. Run each app's check against the working tree before
`scripts/tag`, as AGENTS.md says.
