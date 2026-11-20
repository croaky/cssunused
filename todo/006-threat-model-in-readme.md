# 006: the threat model moves to README.md

Title: `doc: keep the threat model in the README`

Finding: none.

## Need

The threat model names the adversaries and the accepted risks. The
resolution notes of earlier findings cite it. It lived in
`todo/README.md`, and the fix in 6e93b0d deleted it with the old
index. From 6e93b0d to the commit that added this plan, no file at
`main` held the accepted risks that those notes cite. Each new
finding in an accepted class then gets argued again.

## Change

1. Move the "Threat model" section of `todo/README.md` to a new
   "Threat model" section of `README.md`.
2. Leave a line in `todo/README.md` that points to it.
3. Add a line to the "Changes" section of `AGENTS.md`: a change that
   accepts a risk adds it to the threat model in `README.md`.

## Cost

Docs only.
