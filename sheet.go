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
	// IDSelectors is each resolved selector that names an ID, in any
	// layer, in source order. Only the rule that writes the ID counts,
	// not each rule nested under it.
	IDSelectors []string
	// WidthMedia is the head of each @media rule that tests min-width
	// or max-width, in source order.
	WidthMedia []string
	// Important is each !important declaration that the standard does
	// not allow, with its selector, in source order. The standard allows
	// one in the base layer and on the .hidden utility.
	Important []string
}

// frame is one open block: the resolved selector list its children
// resolve against, and the cascade layer it is in.
type frame struct {
	sels  []string
	layer string
}

// widthMediaRE matches a media query that tests a width with the old
// min-width or max-width features.
var widthMediaRE = regexp.MustCompile(`\b(?:min|max)-width\b`)

// ParseSheet resolves nesting and collects the class names out of the
// resolved selectors. It walks the source rather than matching lines
// because a selector can span lines and a comment or a data-URI string
// can contain any punctuation the walk cares about.
func ParseSheet(src string) Sheet {
	defined := map[string]bool{}
	layers := map[string]map[string]bool{}
	var out Sheet

	stack := []frame{{sels: []string{""}}}
	var buf strings.Builder

	// declaration reads the text in buf as a declaration of the open
	// block, which ends at a semicolon or at the block's close.
	declaration := func() {
		decl := strings.Join(strings.Fields(buf.String()), " ")
		if !strings.Contains(decl, "!important") {
			return
		}
		top := stack[len(stack)-1]
		sel := strings.Join(top.sels, ", ")
		if top.layer == "base" || (top.layer == "utilities" && sel == ".hidden") {
			return
		}
		out.Important = append(out.Important, sel+" { "+decl+" }")
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
			// A rule is top-level when no parent rule wraps it, so it
			// resolves against the root's empty selector.
			if layer == "components" && !strings.HasPrefix(head, "@") && slices.Equal(top.sels, []string{""}) {
				out.ComponentsTopLevel = append(out.ComponentsTopLevel, head)
			}
			sels := resolve(top.sels, head)
			for _, s := range sels {
				if namesID(head) && namesID(s) {
					out.IDSelectors = append(out.IDSelectors, s)
				}
				for _, cl := range classesIn(s) {
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
			stack = append(stack, frame{sels: sels, layer: layer})
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

// resolve expands one block's selector text against its parents. An
// at-rule (@media, @font-face) sets no selector of its own, so its
// children resolve against the same parents. A @keyframes block holds
// keyframe selectors (`from`, `50%`), which select no element, so its
// children resolve against none.
func resolve(parents []string, text string) []string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "@keyframes") {
		return nil
	}
	if text == "" || strings.HasPrefix(text, "@") {
		return parents
	}
	var out []string
	for _, sel := range splitCommas(text) {
		sel = strings.Join(strings.Fields(sel), " ")
		if sel == "" {
			continue
		}
		for _, p := range parents {
			switch {
			case strings.Contains(sel, "&"):
				out = append(out, strings.ReplaceAll(sel, "&", p))
			case p == "":
				out = append(out, sel)
			default:
				out = append(out, p+" "+sel)
			}
		}
	}
	if len(out) == 0 {
		return parents
	}
	return out
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

// classesIn pulls class names out of a resolved selector. Attribute
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
