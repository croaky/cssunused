# 005: ParseSheet finds the class set of a layer once for each block

Title: `security: find each layer's class set once per block`

Finding: `find_015Z9z7aWRTgcTUhL8BHih1r`.

## Need

The `ParseSheet` Go function looks up `layers[layer]` twice for each
class. The layer name has no limit, and a map with more than eight
keys hashes the whole name on each lookup. Parse time is then the
name length times the class count. A probe against 0bf0335 took
0.17 s for 200 KB, 0.58 s for 400 KB, and 1.59 s for 800 KB. A sheet
of a few MB can hold one CPU core until the CI job times out.

The threat model in `todo/README.md` accepts this class of risk. The fix
costs a few lines at one site, so it lands anyway, as the fixes of
three earlier findings did.

## Change

1. The `{` branch of `ParseSheet` finds the class set of the layer
   once, when it opens a block, and keeps a pointer to it in `frame`.
2. The loop over the classes of a rule writes to that set and does no
   map lookup by name.
3. The report does not change for any sheet.

## Tests

- A sheet with nine class-bearing layers, one layer name of half the
  sheet, and short class selectors in that layer. Parse it at two
  sizes. The larger sheet must take less than three times as long.
  Use `testing.Benchmark`, or a ratio with a wide margin, so the test
  is not flaky.
- The existing tests of layer lines stay as they are.

## Cost

About five lines and one test.

## Options

- A. The change above. Recommended.
- B. Accept the risk under the threat model, and run
  `claudesec resolve --reason accepted_risk` with a note that names the
  attacker, the consequence, and the boundary.

## Rollout

The fix ships with the next tag. Then run
`claudesec resolve --reason fixed find_015Z9z7aWRTgcTUhL8BHih1r` with
a note that names the commit.
