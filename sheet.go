package main

import (
	"maps"
	"regexp"
	"slices"
	"strings"
)

// Sheet is the class selectors a stylesheet declares, and the rules the
// lint reads off it.
type Sheet struct {
	// Defined is every class name, sorted.
	Defined []string
	// Layers maps a cascade layer to the class names its rules name,
	// sorted.
	Layers map[string][]string
	// Unlayered is the selector of each top-level rule outside every
	// layer, in source order.
	Unlayered []string
	// ComponentsTopLevel is each top-level rule in the components layer,
	// in source order.
	ComponentsTopLevel []string
	// IDSelectors is each selector that names an ID, in any layer, in
	// source order. A nested rule gives its own selector, as the sheet
	// writes it, not the selector of its parent too.
	IDSelectors []string
	// WidthMedia is the head of each @media rule that tests min-width
	// or max-width, in source order.
	WidthMedia []string
	// Important is each !important declaration that the standard does
	// not allow, with the own selector of its rule, in source order. The
	// standard allows one in the base layer and on the top-level .hidden
	// utility.
	Important []string
}

// frame is one open block. sel is the own selector of the nearest
// style rule, as the sheet writes it. inRule tells that a style rule
// wraps the block. nested tells that the rule of sel is itself under
// a style rule. keyframes tells that a @keyframes block wraps it.
type frame struct {
	sel       string
	layer     string
	inRule    bool
	nested    bool
	keyframes bool
}

// widthMediaRE matches a media query that tests a width with the old
// min-width or max-width features.
var widthMediaRE = regexp.MustCompile(`\b(?:min|max)-width\b`)

// ParseSheet collects the class names out of the own selector of each
// rule. It walks the source rather than matching lines because a
// selector can span lines and a comment or a data-URI string can
// contain any punctuation the walk cares about.
//
// It does not expand a nested selector against its parents. A parent's
// classes are recorded at the parent, and & adds no class of its own.
// An expansion into each combination of the selector lists grows
// exponentially with the depth of the nest, so a small sheet could use
// all the memory of the machine.
func ParseSheet(src string) Sheet {
	defined := map[string]bool{}
	layers := map[string]map[string]bool{}
	var out Sheet

	stack := []frame{{}}
	var buf strings.Builder

	// declaration reads the text in buf as a declaration of the open
	// block, which ends at a semicolon or at the block's close.
	declaration := func() {
		decl := strings.Join(strings.Fields(buf.String()), " ")
		if !strings.Contains(decl, "!important") {
			return
		}
		top := stack[len(stack)-1]
		if top.layer == "base" || (top.layer == "utilities" && top.sel == ".hidden" && !top.nested) {
			return
		}
		out.Important = append(out.Important, top.sel+" { "+decl+" }")
	}

	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			if end := strings.Index(src[i+2:], "*/"); end >= 0 {
				i += end + 3
			} else {
				i = len(src)
			}
		case c == '"' || c == '\'':
			j := skipString(src, i)
			buf.WriteString(src[i : j+1])
			i = j
		case c == '{':
			head := strings.Join(strings.Fields(buf.String()), " ")
			top := stack[len(stack)-1]
			layer := top.layer
			if name, ok := strings.CutPrefix(head, "@layer "); ok {
				layer = name
			}
			if layer == "" && len(stack) == 1 {
				out.Unlayered = append(out.Unlayered, head)
			}
			if strings.HasPrefix(head, "@media") && widthMediaRE.MatchString(head) {
				out.WidthMedia = append(out.WidthMedia, head)
			}
			next := top
			next.layer = layer
			isRule := head != "" && !strings.HasPrefix(head, "@") && !top.keyframes
			// A rule is top-level when no parent rule wraps it.
			if layer == "components" && isRule && !top.inRule {
				out.ComponentsTopLevel = append(out.ComponentsTopLevel, head)
			}
			// A @keyframes block holds keyframe selectors (`from`,
			// `50%`), which select no element. An at-rule such as
			// @media sets no selector, so its block keeps the parent's.
			switch {
			case strings.HasPrefix(head, "@keyframes"):
				next.keyframes = true
			case isRule:
				next = frame{sel: head, layer: layer, inRule: true, nested: top.inRule}
				for _, s := range splitCommas(head) {
					if namesID(s) {
						out.IDSelectors = append(out.IDSelectors, strings.TrimSpace(s))
					}
				}
				for _, cl := range classesIn(head) {
					defined[cl] = true
					if layer == "" {
						continue
					}
					if layers[layer] == nil {
						layers[layer] = map[string]bool{}
					}
					layers[layer][cl] = true
				}
			}
			stack = append(stack, next)
			buf.Reset()
		case c == '}':
			declaration()
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
			buf.Reset()
		case c == ';':
			declaration()
			buf.Reset()
		default:
			buf.WriteByte(c)
		}
	}

	out.Defined = sortedKeys(defined)
	out.Layers = map[string][]string{}
	for layer, classes := range layers {
		out.Layers[layer] = sortedKeys(classes)
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	return slices.Sorted(maps.Keys(m))
}

// splitCommas splits a selector list on the commas that separate
// selectors, not the ones inside :is(), :not(), or an attribute value.
func splitCommas(s string) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}

var (
	classRE = regexp.MustCompile(`\.(-?[A-Za-z_][-A-Za-z0-9_]*)`)
	attrRE  = regexp.MustCompile(`\[[^\]]*\]`)
)

// namesID reports whether a selector names an ID. Attribute selectors
// go first, because one can hold a # that starts no ID
// (`[href="#top"]`).
func namesID(sel string) bool {
	return strings.Contains(attrRE.ReplaceAllString(sel, " "), "#")
}

// classesIn pulls class names out of a selector. Attribute
// selectors go first, because one can hold a dot that starts no class
// (`[href$=".pdf"]`).
func classesIn(sel string) []string {
	sel = attrRE.ReplaceAllString(sel, " ")
	var out []string
	for _, m := range classRE.FindAllStringSubmatch(sel, -1) {
		out = append(out, m[1])
	}
	return out
}

// skipString returns the index of the closing quote of the string
// literal opening at i, or the last index if it is unterminated.
//
// A backslash escapes the next byte in every literal here except a Go raw
// string, where it is an ordinary character. Escaping there would step
// over the closing backtick of `C:\` and swallow the rest of the file.
func skipString(s string, i int) int {
	quote := s[i]
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			if quote != '`' {
				j++
			}
		case quote:
			return j
		}
	}
	return len(s) - 1
}
