package main

import (
	"strings"
	"testing"

	"github.com/croaky/is"
)

func TestLintComponentsAndPrint(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	got := Lint(Sheet{Layers: map[string][]string{
		"components": {
			"InvalidUpper",
			"button",
			"button--primary",
			"card_list_item",
			"tip__label--bold",
			"icon_remove--sm",
			"label--bold",
		},
		"print": {"report", "no_print", "print-hide", "report__title"},
	}}, nil)

	is.Eq(got, []string{
		"components: .InvalidUpper: a class here is kebab-case with an optional --modifier; nest an element under its block",
		"components: .card_list_item: a class here is kebab-case with an optional --modifier; nest an element under its block",
		"components: .icon_remove--sm: a class here is kebab-case with an optional --modifier; nest an element under its block",
		"components: .tip__label--bold: a class here is kebab-case with an optional --modifier; nest an element under its block",
		"print: .no_print: a class here is kebab-case with an optional --modifier; nest an element under its block",
		"print: .report__title: a class here is kebab-case with an optional --modifier; nest an element under its block",
	})
}

func TestLintUtilities(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	got := Lint(Sheet{Layers: map[string][]string{
		"utilities": {"bad--util", "bad__util", "flex-1", "mt-1"},
	}}, nil)

	is.Eq(got, []string{
		"utilities: .bad--util: a utility has no __ and no --",
		"utilities: .bad__util: a utility has no __ and no --",
	})
}

func TestLintRejectsAClassOutsideComponentsUtilitiesAndPrint(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	// "component" is a typo for "components". The lint reports its
	// class rather than skip it.
	got := Lint(Sheet{Layers: map[string][]string{
		"base":      {"base-class"},
		"component": {"typo-class"},
	}}, nil)

	is.Eq(got, []string{
		"base: .base-class: only the components, utilities, and print layers define classes",
		"component: .typo-class: only the components, utilities, and print layers define classes",
	})
}

func TestLintRejectsARuleOutsideEveryLayer(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	// A rule with no class counts too: below the layers, it would beat
	// every utility.
	got := Lint(Sheet{Unlayered: []string{".stray", "[app-hook]"}}, nil)

	is.Eq(got, []string{
		"no layer: .stray: a rule outside every layer beats every layer",
		"no layer: [app-hook]: a rule outside every layer beats every layer",
	})
}

func TestLintRejectsNonClassTopLevelInComponents(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	got := Lint(Sheet{
		ComponentsTopLevel: []string{
			"legend",
			"input[type=\"text\"]",
			"a time",
			".card",
			":is(.avatar--small, .list-marker) .avatar",
			":is(h1, h2)",
			"#main_tabs",
		},
	}, nil)

	is.Eq(got, []string{
		"components: #main_tabs: a top-level rule in components must start with a class",
		"components: :is(h1, h2): a top-level rule in components must start with a class",
		"components: a time: a top-level rule in components must start with a class",
		"components: input[type=\"text\"]: a top-level rule in components must start with a class",
		"components: legend: a top-level rule in components must start with a class",
	})
}

// An :is() nest deeper than maxIsDepth fails, so the check of a
// hostile selector stays linear in its length. A nest at the limit
// still passes.
func TestLintLimitsIsDepth(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	nest := func(depth int) string {
		return strings.Repeat(":is(", depth) + ".card" + strings.Repeat(")", depth)
	}
	deep := nest(maxIsDepth + 1)
	got := Lint(Sheet{ComponentsTopLevel: []string{nest(maxIsDepth), deep}}, nil)

	is.Eq(got, []string{
		"components: " + deep + ": a top-level rule in components must start with a class",
	})
}

func TestLintRejectsAnElementNameInAReference(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	// A branch cut before a rename can still write the old element
	// name. The sheet does not define it, so only this check sees it.
	got := Lint(Sheet{}, map[string]bool{"card": true, "table-scroll__handle": true})

	is.Eq(got, []string{
		"reference: .table-scroll__handle: a referenced class has no __",
	})
}

func TestLintRejectsAnIDSelector(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	got := Lint(Sheet{IDSelectors: []string{"#card_title"}}, nil)

	is.Eq(got, []string{
		"id: #card_title: an ID outranks every class, so style a class instead",
	})
}

// A width media query is a range. max-width: 70em and min-width: 70em
// both match at 70em, so a screen that wide gets both layouts.
func TestLintRejectsAWidthMediaQuery(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	got := Lint(Sheet{WidthMedia: []string{"@media (max-width: 48em)"}}, nil)

	is.Eq(got, []string{
		"media: @media (max-width: 48em): write the width as a range, such as (width < 70em)",
	})
}

func TestLintRejectsImportant(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	got := Lint(Sheet{Important: []string{".card { color: red !important }"}}, nil)

	is.Eq(got, []string{
		"important: .card { color: red !important }: !important goes only in base and on .hidden; a later layer wins without it",
	})
}
