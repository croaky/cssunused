package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// componentRE matches a kebab-case name and an optional --modifier.
var componentRE = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*(?:--[a-z0-9-]+)?$`)

// Lint returns one line, sorted, for each class that breaks the naming
// rule of its cascade layer:
//
//   - components and print: a kebab-case name with an optional
//     --modifier, and no __. An element is a nested selector under its
//     block. Every top-level rule in components must start with a
//     class. A nest of :is() deeper than maxIsDepth does not.
//   - utilities: no __ and no --.
//   - every other layer: no class at all.
//
// It also rejects every rule outside a layer, with a class or without.
// Such a rule beats every layered rule, whatever its specificity, so
// one added below the layers would override the utilities. It rejects
// an ID selector in any layer, because an ID outranks every class rule
// in its layer. A referenced class has no __, even when the sheet does
// not define it.
//
// Two rules hold the sheet to its own order. A media query writes its
// width as a range, (width < 70em), because max-width: 70em and
// min-width: 70em both match at 70em, and a screen that wide gets both
// layouts. And !important goes only in base, where it beats a style
// that a browser extension writes inline, and on the .hidden utility,
// which must beat every display utility. A later layer beats an earlier
// one without it. Only a top-level .hidden rule in utilities is that
// utility.
//
// A line about a nested rule names it by its own selector, as the sheet
// writes it, so grep finds it. An important: line and a layer line cut
// the selector or the layer name at maxShown.
func Lint(sheet Sheet, refs map[string]bool) []string {
	var out []string
	for c := range refs {
		if strings.Contains(c, "__") {
			out = append(out, fmt.Sprintf(
				"reference: .%s: a referenced class has no __", c))
		}
	}
	for _, sel := range sheet.Unlayered {
		out = append(out, fmt.Sprintf(
			"no layer: %s: a rule outside every layer beats every layer", sel))
	}
	for _, sel := range sheet.IDSelectors {
		out = append(out, fmt.Sprintf(
			"id: %s: an ID outranks every class, so style a class instead", sel))
	}
	for _, head := range sheet.WidthMedia {
		out = append(out, fmt.Sprintf(
			"media: %s: write the width as a range, such as (width < 70em)", head))
	}
	for _, decl := range sheet.Important {
		out = append(out, fmt.Sprintf(
			"important: %s: !important goes only in base and on .hidden; a later layer wins without it", decl))
	}
	for _, head := range sheet.ComponentsTopLevel {
		for _, sel := range splitCommas(head) {
			sel = strings.TrimSpace(sel)
			if !startsWithClass(sel, 0) {
				out = append(out, fmt.Sprintf(
					"components: %s: a top-level rule in components must start with a class", sel))
			}
		}
	}
	for layer, classes := range sheet.Layers {
		shown := shorten(layer)
		for _, c := range classes {
			switch layer {
			case "components", "print":
				if !componentRE.MatchString(c) {
					out = append(out, fmt.Sprintf(
						"%s: .%s: a class here is kebab-case with an optional --modifier; nest an element under its block", shown, c))
				}
			case "utilities":
				if strings.Contains(c, "__") || strings.Contains(c, "--") {
					out = append(out, fmt.Sprintf(
						"utilities: .%s: a utility has no __ and no --", c))
				}
			default:
				out = append(out, fmt.Sprintf(
					"%s: .%s: only the components, utilities, and print layers define classes", shown, c))
			}
		}
	}
	slices.Sort(out)
	return out
}

// maxShown is the longest selector or layer name that a report line
// repeats. A line for each !important declaration repeats the rule's
// selector, and a line for each class repeats its layer name. Without
// a cut, one long selector and many declarations make lines whose total
// size grows with the square of the sheet.
const maxShown = 80

// shorten cuts s to maxShown bytes, at the start of a rune, and marks
// the cut with "...".
func shorten(s string) string {
	if len(s) <= maxShown {
		return s
	}
	n := maxShown
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + "..."
}

// maxIsDepth is the deepest nest of :is() that startsWithClass reads.
// Each level scans the rest of the selector again, so the limit keeps
// the check linear in the length of the selector.
const maxIsDepth = 8

// startsWithClass reports whether the trimmed selector sel starts with
// a class selector. An :is() selector passes when every selector inside
// it starts with a class. A nest of :is() deeper than maxIsDepth fails.
func startsWithClass(sel string, depth int) bool {
	if strings.HasPrefix(sel, ".") {
		return true
	}
	if !strings.HasPrefix(sel, ":is(") || depth >= maxIsDepth {
		return false
	}
	end := matchParen(sel, len(":is"))
	if end < 0 {
		return false
	}
	for _, arg := range splitCommas(sel[len(":is("):end]) {
		if !startsWithClass(strings.TrimSpace(arg), depth+1) {
			return false
		}
	}
	return true
}

// matchParen returns the index of the closing parenthesis matching the
// open parenthesis at index open, or -1.
func matchParen(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
