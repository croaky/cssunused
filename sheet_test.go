package main

import (
	"strings"
	"testing"

	"github.com/croaky/is"
)

func TestParseSheetResolvesNesting(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const src = `
.card {
  color: red;

  &:hover {
    color: blue;
  }

  .card_title,
  .card_slim {
    margin: 0;
  }

  .drawer & {
    padding: 0;
  }

  @media screen and (width >= 50em) {
    .card_wide {
      margin: 1rem;
    }
  }
}
`
	is.Eq(ParseSheet(src).Defined, []string{"card", "card_slim", "card_title", "card_wide", "drawer"})
}

func TestParseSheetIgnoresNonSelectorDots(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const src = `
/* .block_commented {} */
:root {
  --icon--close: url("data:image/svg+xml,<svg viewBox=\"0 0 20 20\"/>");
}
a[href$=".pdf"] {
  padding: 0.5rem;
}
.icon-btn {
  svg path,
  svg circle {
    width: 1rem;
  }
}
`
	is.Eq(ParseSheet(src).Defined, []string{"icon-btn"})
}

func TestParseSheetLayers(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const src = `
@layer reset, tokens, base, components, utilities;

@layer reset {
  *, *::before { box-sizing: border-box; }
}

@layer tokens {
  :root { --font--body: sans-serif; }
}

@layer base {
  h1 { margin: 0; }
}

@layer components {
  .card {
    color: red;
    .title { font-weight: bold; }
    &.card--featured { border: 1px solid blue; }
  }
}

@layer utilities {
  .mt-1 { margin-top: 1rem; }
  .flex { display: flex; }
}

.stray { color: red; }

body { margin: 0; }
`
	sheet := ParseSheet(src)
	// reset, tokens, and base name no class, so they have no key.
	is.Eq(sheet.Layers, map[string][]string{
		"components": {"card", "card--featured", "title"},
		"utilities":  {"flex", "mt-1"},
	})
	is.Eq(sheet.Unlayered, []string{".stray", "body"})
	is.Eq(sheet.ComponentsTopLevel, []string{".card"})
	is.Eq(sheet.Defined, []string{"card", "card--featured", "flex", "mt-1", "stray", "title"})
}

func TestParseSheetComponentsTopLevel(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const src = `
@layer reset, tokens, base, components, utilities;

@layer components {
  legend {
    color: red;
  }

  .card {
    color: blue;

    .title {
      font-weight: bold;
    }
  }

  @keyframes pulse {
    0% { opacity: 0; }
  }

  @media (width >= 50em) {
    figure {
      margin: 0;
    }
  }
}
`
	// A rule inside @media is still top-level: it starts with no parent.
	is.Eq(ParseSheet(src).ComponentsTopLevel, []string{"legend", ".card", "figure"})
}

func TestParseSheetIDSelectors(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const src = `
@layer reset, tokens, base, components, utilities;

@layer base {
  a[href="#top"] { color: red; }
}

@layer components {
  #main_tabs { position: relative; }

  .card {
    color: #333;

    #card_title { font-weight: bold; }
  }

  .rows:has(+ #more_rows) { padding: 0; }
}
`
	// An attribute value and a hex color name no ID.
	// A nested rule is named by its own selector, as the sheet writes
	// it, so grep finds it.
	is.Eq(ParseSheet(src).IDSelectors, []string{"#main_tabs", "#card_title", ".rows:has(+ #more_rows)"})
}

// A media query that tests min-width or max-width is listed, at the top
// level and nested. A range and a query on another feature are not.
func TestParseSheetWidthMedia(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const src = `
@layer components {
  .card {
    @media screen and (min-width: 50em) {
      margin: 0;
    }
  }

  @media (max-width: 48em) {
    .card { padding: 0; }
  }

  @media (width < 70em) {
    .card { padding: 0; }
  }

  @media (prefers-color-scheme: dark) {
    .card { color: white; }
  }
}
`
	is.Eq(ParseSheet(src).WidthMedia, []string{"@media screen and (min-width: 50em)", "@media (max-width: 48em)"})
}

// !important passes in base and on the .hidden utility, and is listed
// anywhere else, with the last declaration of a block that has no
// closing semicolon too. A comment that names it is not a declaration.
func TestParseSheetImportant(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const src = `
@layer base {
  input[data-extension] { display: none !important; }
}

@layer components {
  /* No !important here. */
  .card {
    color: red !important;

    .title { margin: 0 !important }
  }
}

@layer utilities {
  .hidden { display: none !important; }
  .flex { display: flex !important; }
  .menu {
    .hidden { display: block !important; }
  }
}

.stray { color: blue !important; }
`
	// A nested rule is named by its own selector. A nested .hidden is
	// not the .hidden utility, so it is listed.
	is.Eq(ParseSheet(src).Important, []string{
		".card { color: red !important }",
		".title { margin: 0 !important }",
		".flex { display: flex !important }",
		".hidden { display: block !important }",
		".stray { color: blue !important }",
	})
}

// A @layer block nested in a style rule puts only the classes of its
// own rules in that layer, not the classes of the parent rule.
func TestParseSheetNestedLayer(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const src = `
@layer components {
  .card {
    @layer utilities {
      .mt-1 { margin-top: 1rem; }
    }
  }
}
`
	is.Eq(ParseSheet(src).Layers, map[string][]string{
		"components": {"card"},
		"utilities":  {"mt-1"},
	})
}

// A deep nest with a selector list and & at each level parses in time
// linear in the size of the sheet. An expansion of the nest into every
// combination of its selectors would hold 3^40 selectors.
func TestParseSheetDeepNestingIsLinear(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const depth = 40
	src := strings.Repeat(".a, & .b, & & {", depth) + "color: red !important;" + strings.Repeat("}", depth)
	sheet := ParseSheet(src)

	is.Eq(sheet.Defined, []string{"a", "b"})
	is.Eq(sheet.Important, []string{".a, & .b, & & { color: red !important }"})
}
