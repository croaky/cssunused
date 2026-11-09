package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// skipDirs are directories whose contents cannot vouch for a selector.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"testdata":     true,
	"vendor":       true,
}

// Refs collects the class names the app can put on an element, read from
// the positions that put one there: a selector chain or a class attribute
// in a template, a className, a classList call, a class selector, or a
// class-taking builder in a script, and a class-named literal in Go.
// builders names the script functions whose argument list holds a class.
//
// Reading positions rather than words is what lets a dead selector be
// reported. A scan of every word in every template and string literal
// lets any English word vouch for a selector of the same name: .example,
// .hint, .indicator, and .tag are all ordinary words, and none of them
// could ever be reported.
//
// The cost is that a class assembled at run time is invisible here. That
// is the point. The answer is to spell the name in source, in a lookup
// that returns whole literals, rather than widen the scan until nothing
// can be reported.
func Refs(root string, builders []string) (map[string]bool, error) {
	builderRE := builderPattern(builders)
	paths, err := sourceFiles(root)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, path := range paths {
		ext := filepath.Ext(path)
		switch ext {
		case ".hml", ".html", ".go", ".ts", ".tsx", ".js":
		default:
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		text := string(b)
		switch ext {
		case ".hml":
			// A template names a class in a selector chain or a class
			// attribute, and its `:javascript` block names one however
			// a script does, so both readings run over the whole file.
			add(out, hmlClasses(text))
			add(out, scriptClasses(text, builderRE))
		case ".html":
			add(out, attrClasses(text))
		case ".ts", ".tsx", ".js":
			add(out, scriptClasses(text, builderRE))
		case ".go":
			classes, err := goClasses(path, text)
			if err != nil {
				return nil, err
			}
			add(out, classes)
		}
	}
	return out, nil
}

// sourceFiles lists the files under root that can name a class.
//
// In a git checkout it lists what git does: the tracked files and the
// untracked ones that .gitignore does not cover. A build writes its
// output to an ignored directory, and a stale bundle there still names
// the classes of an old version of the app. A new file counts before
// git add, so a check run before a commit sees it. Outside a checkout,
// such as a module in the module cache, it lists every file.
func sourceFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	cmd.Dir = root
	if out, err := cmd.Output(); err == nil {
		var paths []string
		for rel := range bytes.SplitSeq(bytes.TrimSuffix(out, []byte{0}), []byte{0}) {
			if len(rel) == 0 || inSkipDir(string(rel)) {
				continue
			}
			path := filepath.Join(root, string(rel))
			// git lists a tracked file that the work tree deleted.
			if _, err := os.Stat(path); err == nil {
				paths = append(paths, path)
			}
		}
		return paths, nil
	}
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// inSkipDir reports whether a slash-separated relative path sits in one
// of skipDirs.
func inSkipDir(rel string) bool {
	parts := strings.Split(rel, "/")
	for _, dir := range parts[:len(parts)-1] {
		if skipDirs[dir] {
			return true
		}
	}
	return false
}

func add(out map[string]bool, classes []string) {
	for _, c := range classes {
		if c != "" {
			out[c] = true
		}
	}
}

// name is a class name: a CSS identifier, which can lead with a hyphen
// (`-mt-h` is a negative margin utility).
const name = `-?[A-Za-z_][A-Za-z0-9_-]*`

var (
	// chainRE matches the selector at the head of an hml line: an
	// optional tag, then the `.class` and `#id` chain hung off it, as in
	// `%header.card-header.flex`, `.mb-1.flex-2`, or `#team_warning.hidden`.
	chainRE = regexp.MustCompile(`(?m)^[ \t]*(?:[=-][ \t]+)?(?:%[A-Za-z][A-Za-z0-9]*)?((?:[.#]` + name + `)+)`)
	// nameRE picks the names out of one chain, dropping the `#id` ones.
	nameRE = regexp.MustCompile(`([.#])(` + name + `)`)
	// classAttrRE matches a class attribute's literal: hml's `class: "..."`,
	// which covers `%span{ class: "tag" }` and a mount's
	// `html: {class: "typeahead"}`, and html's `class="..."`. `klass` is the
	// same thing under the name a partial's local wears, since `class` is
	// the attribute it ends up in.
	classAttrRE = regexp.MustCompile(`(?:class|klass)\s*[:=]\s*("[^"]*"|'[^']*')`)
	// classNameRE matches a className assignment or property.
	classNameRE = regexp.MustCompile("className\\s*[:=]\\s*(\"[^\"]*\"|'[^']*'|`[^`]*`)")
	// classListRE matches the whole argument list of a classList call, so
	// every name in an add("a", "b") lands.
	classListRE = regexp.MustCompile(`classList\.[A-Za-z]+\(([^)]*)\)`)
	// selectorRE matches the selector string handed to a DOM query. A
	// generic type argument can sit between the name and the paren, as in
	// querySelector<HTMLElement>(".drawer").
	selectorRE = regexp.MustCompile("(?:querySelectorAll|querySelector|closest|matches)\\s*(?:<[^<>()]*>)?\\s*\\(\\s*(\"[^\"]*\"|'[^']*'|`[^`]*`)")
	// litRE matches a quoted literal, for the argument lists captured whole.
	litRE = regexp.MustCompile("\"[^\"]*\"|'[^']*'|`[^`]*`")
	// classInSelectorRE matches one `.class` inside a selector string.
	classInSelectorRE = regexp.MustCompile(`\.(` + name + `)`)
	// interpRE matches the interpolations a class string can carry, in hml
	// (`#{...}`) and in a template literal (`${...}`). The names around
	// one still count; the hole itself names nothing.
	interpRE = regexp.MustCompile(`[#$]\{[^}]*\}`)
)

// builderPattern matches a call to one of the element builders that take
// a class string, such as `div("row")`. A widget built through them names
// its classes there and nowhere else. The argument list is captured whole,
// because the class can be picked by a conditional inside the call. No
// builders means no pattern.
func builderPattern(builders []string) *regexp.Regexp {
	if len(builders) == 0 {
		return nil
	}
	quoted := make([]string, 0, len(builders))
	for _, b := range builders {
		quoted = append(quoted, regexp.QuoteMeta(b))
	}
	return regexp.MustCompile(`\b(?:` + strings.Join(quoted, "|") + `)\(([^)]*)\)`)
}

// hmlClasses reads the class names a template writes into markup: the
// chain at the head of a line, and the literal part of a class attribute.
func hmlClasses(src string) []string {
	var out []string
	for _, m := range chainRE.FindAllStringSubmatch(src, -1) {
		for _, name := range nameRE.FindAllStringSubmatch(m[1], -1) {
			if name[1] == "." {
				out = append(out, name[2])
			}
		}
	}
	return append(out, attrClasses(src)...)
}

// attrClasses reads the class names out of every class attribute.
func attrClasses(src string) []string {
	var out []string
	for _, m := range classAttrRE.FindAllStringSubmatch(src, -1) {
		out = append(out, words(m[1])...)
	}
	return out
}

// scriptClasses reads the class names a script names: a className, the
// arguments of a classList call, the class selectors in a DOM query, and
// the class argument of an element builder.
func scriptClasses(src string, builderRE *regexp.Regexp) []string {
	var out []string
	for _, m := range classNameRE.FindAllStringSubmatch(src, -1) {
		out = append(out, words(m[1])...)
	}
	calls := classListRE.FindAllStringSubmatch(src, -1)
	if builderRE != nil {
		calls = append(calls, builderRE.FindAllStringSubmatch(src, -1)...)
	}
	for _, m := range calls {
		for _, lit := range litRE.FindAllString(m[1], -1) {
			out = append(out, words(lit)...)
		}
	}
	for _, m := range selectorRE.FindAllStringSubmatch(src, -1) {
		for _, name := range classInSelectorRE.FindAllStringSubmatch(m[1], -1) {
			out = append(out, name[1])
		}
	}
	return out
}

// goClasses reads the class names a Go file writes: the literals a
// function named for a class returns, the literal a template local named
// for a class holds, and the literal a variable or a field named for a
// class is assigned. Handlers hand a class to a template one of those
// ways.
func goClasses(path, src string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
	if err != nil {
		return nil, err
	}
	var out []string
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Body == nil || !isClassName(node.Name.Name) {
				return true
			}
			ast.Inspect(node.Body, func(inner ast.Node) bool {
				if lit, ok := inner.(*ast.BasicLit); ok {
					out = append(out, words(lit.Value)...)
				}
				return true
			})
		case *ast.KeyValueExpr:
			lit, ok := node.Value.(*ast.BasicLit)
			if !ok {
				return true
			}
			switch key := node.Key.(type) {
			case *ast.BasicLit:
				if isClassName(key.Value) {
					out = append(out, words(lit.Value)...)
				}
			case *ast.Ident:
				if isClassName(key.Name) {
					out = append(out, words(lit.Value)...)
				}
			}
		case *ast.AssignStmt:
			for i, lhs := range node.Lhs {
				if i >= len(node.Rhs) {
					break
				}
				// A variable (mainClass := ...) or a field set through a
				// selector (sc.Class = ...).
				var lhsName string
				switch l := lhs.(type) {
				case *ast.Ident:
					lhsName = l.Name
				case *ast.SelectorExpr:
					lhsName = l.Sel.Name
				}
				if !isClassName(lhsName) {
					continue
				}
				if lit, ok := node.Rhs[i].(*ast.BasicLit); ok {
					out = append(out, words(lit.Value)...)
				}
			}
		}
		return true
	})
	return out, nil
}

// isClassName reports whether a Go name says its value is a class: a
// function like StatusClass, a map key like "status_dot_class", a struct
// field like StatusClasses.
func isClassName(name string) bool {
	return strings.Contains(strings.ToLower(name), "class")
}

// words splits a quoted class string into names, dropping the quotes and
// any interpolation hole.
func words(lit string) []string {
	if unquoted, err := strconv.Unquote(lit); err == nil {
		lit = unquoted
	} else {
		lit = strings.Trim(lit, "\"'`")
	}
	return strings.Fields(interpRE.ReplaceAllString(lit, " "))
}
