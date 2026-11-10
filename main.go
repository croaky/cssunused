// Command cssunused reports class selectors in an app's stylesheets that
// nothing renders, and the rules of the sheet that break its layered
// standard (see Lint).
//
// A stylesheet only grows when nothing distinguishes a live selector
// from a dead one. This makes that distinction checkable: it resolves
// each sheet into a flat list of class selectors, collects the names the
// app puts on an element, and prints the difference.
//
// It reads class positions, not words, so a name assembled at run time
// is invisible to it and reported as dead. There is no allowlist for
// that: every class the app renders is spelled out in a template, in a
// script, or in a Go lookup keyed by the value that used to be
// interpolated into the name. An allowlist is how a whole utility scale
// gets deleted as unused while a Sprintf goes on naming it.
//
// An app that serves the sheet of a Go module beside its own, or that
// renders the markup of a Go module, names the module with -module. The
// command then reads the module too: its .css files for the classes it
// defines, and its views, scripts, and Go for the classes it writes. A
// module rule that only the module's views write is not unused, and an
// app rule that styles a module class is live when the module writes
// that class. The lint reads the app's sheets only, because a module
// keeps the rules of its own repo.
//
// Usage:
//
//	cssunused -sheet ui/css/app.css [-sheet ...] [-root .] [-module path ...] [-builder name ...]
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// list is a flag that can repeat.
type list []string

func (l *list) String() string { return strings.Join(*l, ",") }

func (l *list) Set(v string) error {
	*l = append(*l, v)
	return nil
}

func main() {
	log.SetFlags(0)

	var sheets, modules, builders list
	root := flag.String("root", ".", "directory of the app's templates, scripts, and Go")
	flag.Var(&sheets, "sheet", "an app stylesheet to check; repeat for each sheet")
	flag.Var(&modules, "module", "a required Go module whose sheet the app serves; repeat for each module")
	flag.Var(&builders, "builder", "a script function whose arguments hold a class; repeat for each")
	flag.Parse()
	if len(sheets) == 0 {
		log.Fatal("cssunused: name one or more -sheet files")
	}

	issues, err := run(*root, sheets, modules, builders)
	if err != nil {
		log.Fatalf("cssunused: %v", err)
	}
	for _, line := range issues {
		fmt.Fprintln(os.Stderr, line)
	}
	if len(issues) > 0 {
		fmt.Fprintf(os.Stderr, "cssunused: %d issue(s)\n", len(issues))
		os.Exit(1)
	}
}

// run returns one line for each unused class and each lint violation,
// unused classes first.
func run(root string, sheets, modules, builders []string) ([]string, error) {
	refs, err := Refs(root, builders)
	if err != nil {
		return nil, err
	}
	var defined, violations []string
	// A class with __ in the app's markup fails once, not once for each
	// sheet.
	for _, v := range Lint(Sheet{}, refs) {
		violations = append(violations, "lint: "+v)
	}
	for _, path := range sheets {
		sheet, err := readSheet(path)
		if err != nil {
			return nil, err
		}
		defined = append(defined, sheet.Defined...)
		for _, v := range Lint(sheet, nil) {
			violations = append(violations, "lint: "+path+": "+v)
		}
	}
	for _, module := range modules {
		dir, err := moduleDir(root, module)
		if err != nil {
			return nil, err
		}
		classes, moduleRefs, err := readModule(dir, builders)
		if err != nil {
			return nil, err
		}
		maps.Copy(refs, moduleRefs)
		defined = append(defined, classes...)
	}
	var out []string
	for _, c := range slices.Compact(slices.Sorted(slices.Values(defined))) {
		if !refs[c] {
			out = append(out, "unused: ."+c)
		}
	}
	return append(out, violations...), nil
}

// readSheet parses the stylesheet at path.
func readSheet(path string) (Sheet, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return Sheet{}, err
	}
	sheet := ParseSheet(string(src))
	// A sheet with no classes means the file moved or the parse broke.
	// Reporting nothing would read as a pass.
	if len(sheet.Defined) == 0 {
		return Sheet{}, fmt.Errorf("no class selectors in %s", path)
	}
	return sheet, nil
}

// readModule reads the module in dir: the classes its sheets define, and
// the classes its views, scripts, and Go write.
//
// A module can ship no sheet. A package that writes markup can leave
// the colors to each app, and the app's sheet styles the classes the
// package writes. But a module that neither defines nor writes a class
// is an error, because the flag then names the wrong module.
//
// It reads every file of the module and starts no git. The directory is
// a copy in the module cache, or a replace target that the app's go.mod
// names, so the files of the change under check can choose it.
func readModule(dir string, builders []string) ([]string, map[string]bool, error) {
	classes, err := moduleClasses(dir)
	if err != nil {
		return nil, nil, err
	}
	paths, err := walkFiles(dir)
	if err != nil {
		return nil, nil, err
	}
	refs, err := readRefs(paths, builders)
	if err != nil {
		return nil, nil, err
	}
	if len(classes) == 0 && len(refs) == 0 {
		return nil, nil, errors.New("no class defined or written in " + dir)
	}
	return classes, refs, nil
}

// moduleClasses is every class that a .css file of the module defines.
func moduleClasses(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		if d.IsDir() || filepath.Ext(path) != ".css" {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out = append(out, ParseSheet(string(src)).Defined...)
		return nil
	})
	return out, err
}

// moduleDir is the directory of a module the app requires: the replace
// target in a local experiment, or its copy in the module cache.
func moduleDir(root, module string) (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", module)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go list -m %s: %w", module, err)
	}
	dir := strings.TrimSpace(string(out))
	// An empty Dir means the module is not downloaded, and reading no
	// module would report each of its rules as unused.
	if dir == "" {
		return "", fmt.Errorf("go list -m %s: no source directory; run go mod download", module)
	}
	return dir, nil
}
