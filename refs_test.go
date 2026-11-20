package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/croaky/is"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	is := is.New(t)
	is.NoErr(os.MkdirAll(filepath.Dir(path), 0o755))
	is.NoErr(os.WriteFile(path, []byte(content), 0o644))
}

func TestRefsReadsClassPositions(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()

	write(t, filepath.Join(root, "ui", "views", "a.hml"),
		"%header#top.card_header.flex\n"+
			"  %span{ class: \"tag #{data.status_class}\" }\n"+
			"  .-mt-h\n"+
			"  = render \"reports/cell\", klass: \"h-25\"\n")
	write(t, filepath.Join(root, "ui", "views", "svgs", "_logo.html"), `<svg class="logo_icon"></svg>`)
	write(t, filepath.Join(root, "ui", "js", "app.ts"),
		"el.classList.add(\"drawer_toggle\", \"active\");\n"+
			"const row = div(\"predicate_row\");\n"+
			"row.className = \"card\";\n"+
			"document.querySelector<HTMLElement>(\".drawer\");\n")
	write(t, filepath.Join(root, "static", "script.js"),
		"document.querySelectorAll(\".log-line\").forEach((el) => el.classList.toggle(\"open\"));\n")
	write(t, filepath.Join(root, "web", "handler.go"),
		"package web\n\n"+
			"func statusClass() string { return \"tag tag-pool\" }\n\n"+
			"func data() map[string]any {\n"+
			"\tmainClass := \"no-nav\"\n"+
			"\tvar sc struct{ Class string }\n"+
			"\tsc.Class = \"chart-bars--long\"\n"+
			"\treturn map[string]any{\"main_class\": mainClass, \"button_class\": \"button\", \"title\": \"Pipeline\"}\n"+
			"}\n")

	refs, err := Refs(root, []string{"div"})
	is.NoErr(err)
	for _, c := range []string{
		// hml: the chain at the head of a line, a class attribute around
		// an interpolation, a negative-margin utility, a klass local.
		"card_header", "flex", "tag", "-mt-h", "h-25",
		// html: a class attribute.
		"logo_icon",
		// TypeScript: a classList call, a builder, a className, a selector.
		"drawer_toggle", "active", "predicate_row", "card", "drawer",
		// JavaScript: a selector and a classList call.
		"log-line", "open",
		// Go: a function named for a class, a variable named for one, a
		// field set through a selector, a map key naming one.
		"tag-pool", "no-nav", "chart-bars--long", "button",
	} {
		is.True(refs[c])
	}
	// An id is not a class, an interpolation names nothing, and a literal
	// in no class position vouches for no selector.
	is.True(!refs["top"])
	is.True(!refs["data"])
	is.True(!refs["Pipeline"])
}

// A function call names a class only when the app names the function as
// a builder.
func TestRefsReadsBuildersTheAppNames(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()

	write(t, filepath.Join(root, "app.ts"), "cell(\"grid-cell\");\nlog(\"grid-row\");\n")

	refs, err := Refs(root, nil)
	is.NoErr(err)
	is.True(!refs["grid-cell"])

	refs, err = Refs(root, []string{"cell"})
	is.NoErr(err)
	is.True(refs["grid-cell"])
	is.True(!refs["grid-row"])
}

// A const or a var named for a class counts, at the top of a file and
// in a function, so a package that writes its classes from named
// values vouches for them. A const named for something else does not.
func TestRefsReadsClassConstsAndVars(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()

	write(t, filepath.Join(root, "token.go"), "package token\n\n"+
		"const (\n\tkeywordClass = \"k\"\n\tstringClass, numberClass = \"s\", \"m\"\n\tmaxDepth = \"deep\"\n)\n\n"+
		"var diffClass = \"diff\"\n\n"+
		"func f() { var lineClass = \"gu\"; _ = lineClass }\n")

	refs, err := Refs(root, nil)
	is.NoErr(err)
	for _, c := range []string{"k", "s", "m", "diff", "gu"} {
		is.True(refs[c])
	}
	is.True(!refs["deep"])
}

func TestRefsReadsNoClassOutOfProse(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()

	// Each of these words is a class name elsewhere in an app. In prose
	// and in an unrelated Go literal none of them names an element, which
	// is what a word scan could not tell.
	write(t, filepath.Join(root, "ui", "views", "a.hml"),
		"%p\n  For example, a hint in the indicator column.\n")
	write(t, filepath.Join(root, "web", "handler.go"),
		"package web\n\nfunc f() string { return \"tag\" }\n")

	refs, err := Refs(root, nil)
	is.NoErr(err)
	for _, c := range []string{"example", "hint", "indicator", "tag"} {
		is.True(!refs[c])
	}
}

// In a git checkout, a file that .gitignore covers names no class. A
// stale bundle in a build directory names the classes of an old
// version of the app. An untracked file that git does not ignore
// counts, so a check before git add sees a new template.
func TestRefsReadsTheFilesGitLists(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()

	git := exec.Command("git", "init", "-q")
	git.Dir = root
	is.NoErr(git.Run())
	write(t, filepath.Join(root, ".gitignore"), "public/\n")
	write(t, filepath.Join(root, "public", "js", "app-1.js"), "el.classList.add(\"old__name\");\n")
	write(t, filepath.Join(root, "ui", "views", "new.hml"), ".new-card\n")

	refs, err := Refs(root, nil)
	is.NoErr(err)
	is.True(refs["new-card"])
	is.True(!refs["old__name"])

	// The default -root is ".", a relative path. git gets the same list.
	wd, err := os.Getwd()
	is.NoErr(err)
	rel, err := filepath.Rel(wd, root)
	is.NoErr(err)
	refs, err = Refs(rel, nil)
	is.NoErr(err)
	is.True(refs["new-card"])
	is.True(!refs["old__name"])
}

// plantRepo writes the files that make dir a repository to git: HEAD,
// objects, refs, and a config whose core.fsmonitor helper writes a
// marker file. A change or a module zip can hold each of these files.
// It returns the path of the marker.
func plantRepo(t *testing.T, dir string) string {
	t.Helper()
	marker := filepath.Join(t.TempDir(), "helper-ran")
	helper := filepath.Join(t.TempDir(), "helper.sh")
	write(t, helper, "#!/bin/sh\ntouch '"+marker+"'\n")
	is.New(t).NoErr(os.Chmod(helper, 0o755))
	write(t, filepath.Join(dir, "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(dir, "objects", "info", "packs"), "")
	write(t, filepath.Join(dir, "refs", "heads", ".keep"), "")
	write(t, filepath.Join(dir, "config"),
		"[core]\n\trepositoryformatversion = 0\n\tbare = false\n\tworktree = .\n\tfsmonitor = "+helper+"\n")
	return marker
}

// A root with no .git entry is not a checkout, so git does not start
// there, whatever repository its files make up. Every file counts.
func TestRefsStartsNoGitOutsideACheckout(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()

	marker := plantRepo(t, root)
	write(t, filepath.Join(root, "app.ts"), "el.classList.add(\"walked\");\n")

	refs, err := Refs(root, nil)
	is.NoErr(err)
	is.True(refs["walked"])
	_, err = os.Stat(marker)
	is.True(os.IsNotExist(err))
}

// A .git entry that git rejects fails the run, and git does not read
// root as a bare repository. The change under check can supply the
// HEAD, objects, refs, and config of that repository, and a
// core.fsmonitor key in the config names a program that git runs.
func TestRefsStartsNoGitOnARejectedGitEntry(t *testing.T) {
	t.Parallel()
	for name, plantGit := range map[string]func(t *testing.T, path string){
		"empty directory": func(t *testing.T, path string) {
			is.New(t).NoErr(os.Mkdir(path, 0o755))
		},
		"dangling symlink": func(t *testing.T, path string) {
			is.New(t).NoErr(os.Symlink(filepath.Join(t.TempDir(), "gone"), path))
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)
			root := t.TempDir()

			marker := plantRepo(t, root)
			plantGit(t, filepath.Join(root, ".git"))
			write(t, filepath.Join(root, "app.ts"), "el.classList.add(\"card\");\n")

			_, err := Refs(root, nil)
			is.HasErr(err)
			_, err = os.Stat(marker)
			is.True(os.IsNotExist(err))
		})
	}
}

// A worktree from git worktree add holds a .git file, not a directory.
// The run lists the files that git lists there, as in a clone.
func TestRefsReadsTheFilesGitListsInAWorktree(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	main := t.TempDir()
	worktree := filepath.Join(t.TempDir(), "wt")

	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "init"},
		{"worktree", "add", "-q", worktree},
	} {
		git := exec.Command("git", args...)
		git.Dir = main
		is.NoErr(git.Run())
	}
	write(t, filepath.Join(worktree, ".gitignore"), "public/\n")
	write(t, filepath.Join(worktree, "public", "app-1.js"), "el.classList.add(\"old__name\");\n")
	write(t, filepath.Join(worktree, "new.hml"), ".new-card\n")

	refs, err := Refs(worktree, nil)
	is.NoErr(err)
	is.True(refs["new-card"])
	is.True(!refs["old__name"])
}

// A module directory is never a checkout of the app, so git does not
// start there, even when the module holds a .git entry.
func TestReadModuleStartsNoGit(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	dir := t.TempDir()

	marker := plantRepo(t, dir)
	write(t, filepath.Join(dir, ".git"), "gitdir: .\n")
	write(t, filepath.Join(dir, "html.go"), "package hl\n\nconst keywordClass = \"k\"\n")

	_, refs, err := readModule(dir, nil)
	is.NoErr(err)
	is.True(refs["k"])
	_, err = os.Stat(marker)
	is.True(os.IsNotExist(err))
}

// In a checkout, a git failure is an error. A walk in its place would
// read the ignored build output that the git list leaves out.
func TestRefsRejectsACheckoutGitCannotList(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()

	write(t, filepath.Join(root, ".git"), "")
	write(t, filepath.Join(root, "app.ts"), "el.classList.add(\"card\");\n")

	_, err := Refs(root, nil)
	is.HasErr(err)
}

// outsideLink writes a file outside every scanned directory, and a
// symlink to it at link. The change under check can hold the symlink,
// since git stores its target as content.
func outsideLink(t *testing.T, link, content string) {
	t.Helper()
	target := filepath.Join(t.TempDir(), "secret")
	write(t, target, content)
	is := is.New(t)
	is.NoErr(os.MkdirAll(filepath.Dir(link), 0o755))
	is.NoErr(os.Symlink(target, link))
}

// A symlink is not read, in the walk and in the files git lists, so a
// file outside the root vouches for no class.
func TestRefsReadsNoSymlink(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	walked := t.TempDir()
	outsideLink(t, filepath.Join(walked, "app.ts"), "el.classList.add(\"secret\");\n")
	refs, err := Refs(walked, nil)
	is.NoErr(err)
	is.True(!refs["secret"])

	listed := t.TempDir()
	git := exec.Command("git", "init", "-q")
	git.Dir = listed
	is.NoErr(git.Run())
	outsideLink(t, filepath.Join(listed, "app.ts"), "el.classList.add(\"secret\");\n")
	refs, err = Refs(listed, nil)
	is.NoErr(err)
	is.True(!refs["secret"])
}

// A symlink in a module is not read, as a sheet or as a source file.
func TestReadModuleReadsNoSymlink(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	dir := t.TempDir()

	write(t, filepath.Join(dir, "lib.css"), "@layer components { .lib-tip { color: red; } }\n")
	outsideLink(t, filepath.Join(dir, "leak.css"), ".leak { color: red; }\n")
	outsideLink(t, filepath.Join(dir, "leak.go"), "package leak\n\nconst leakClass = \"leak-ref\"\n")

	classes, refs, err := readModule(dir, nil)
	is.NoErr(err)
	is.Eq(classes, []string{"lib-tip"})
	is.True(!refs["leak-ref"])
}

// A -sheet that is a symlink is an error, because the change under
// check can replace the sheet with a link to a device or to a file
// outside the checkout. A symlinked parent directory or a sheet outside
// -root also fails.
func TestReadSheetRejectsASymlink(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()
	path := filepath.Join(root, "app.css")

	outsideLink(t, path, ".card { color: red; }\n")

	_, err := run(root, []string{path}, nil, nil, nil)
	is.HasErr(err)
}

func TestRunRejectsSymlinkParentOfSheet(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()
	outside := t.TempDir()

	write(t, filepath.Join(outside, "css", "app.css"), "@layer components { .secret-sheet { color: red; } }\n")
	is.NoErr(os.Symlink(outside, filepath.Join(root, "ui")))
	write(t, filepath.Join(root, "index.hml"), ".card\n")

	got, err := run(root, []string{filepath.Join(root, "ui", "css", "app.css")}, nil, nil, nil)
	is.HasErr(err)
	is.Eq(len(got), 0)
}

func TestRunRejectsSheetOutsideRoot(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()
	outside := t.TempDir()

	sheet := filepath.Join(outside, "app.css")
	write(t, sheet, "@layer components { .outside-sheet { color: red; } }\n")
	write(t, filepath.Join(root, "index.hml"), ".card\n")

	got, err := run(root, []string{sheet}, nil, nil, nil)
	is.HasErr(err)
	is.Eq(len(got), 0)
}

func TestRefsRejectsSymlinkRoot(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	target := t.TempDir()

	git := exec.Command("git", "init", "-q")
	git.Dir = target
	is.NoErr(git.Run())
	write(t, filepath.Join(target, "app.ts"), "el.classList.add(\"outside-repo\");\n")

	link := filepath.Join(t.TempDir(), "web")
	is.NoErr(os.Symlink(target, link))

	refs, err := Refs(link, nil)
	is.HasErr(err)
	is.True(!refs["outside-repo"])
}

func TestRefsRejectsSymlinkParentOfRelativeRoot(t *testing.T) {
	is := is.New(t)
	checkout := t.TempDir()
	outside := t.TempDir()

	write(t, filepath.Join(outside, "web", "app.ts"), "el.classList.add(\"outside-repo\");\n")
	is.NoErr(os.Symlink(outside, filepath.Join(checkout, "apps")))

	write(t, filepath.Join(checkout, "real", "web", "app.ts"), "el.classList.add(\"inside-app\");\n")
	is.NoErr(os.Symlink(filepath.Join(checkout, "real"), filepath.Join(checkout, "alias")))

	t.Chdir(checkout)

	refs, err := Refs(filepath.Join("apps", "web"), nil)
	is.HasErr(err)
	is.True(!refs["outside-repo"])

	refs, err = Refs(filepath.Join("alias", "web"), nil)
	is.HasErr(err)
	is.True(!refs["inside-app"])

	refs, err = Refs(filepath.Join("real", "web"), nil)
	is.NoErr(err)
	is.True(refs["inside-app"])
}

func TestRefsSkipsGitListedFileThroughSymlinkDir(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()
	outside := t.TempDir()

	write(t, filepath.Join(outside, "file.ts"), "el.classList.add(\"escaped-dir\");\n")
	write(t, filepath.Join(root, "dir", "file.ts"), "el.classList.add(\"inside\");\n")
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "dir/file.ts"},
	} {
		git := exec.Command("git", args...)
		git.Dir = root
		is.NoErr(git.Run())
	}
	// Replace the tracked directory on disk with a symlink outside root.
	is.NoErr(os.RemoveAll(filepath.Join(root, "dir")))
	is.NoErr(os.Symlink(outside, filepath.Join(root, "dir")))

	refs, err := Refs(root, nil)
	is.NoErr(err)
	is.True(!refs["escaped-dir"])
}

func TestModuleDirRejectsOutsideReplaceUnlessAllowed(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()
	outside := t.TempDir()

	write(t, filepath.Join(outside, "go.mod"), "module example.com/ext\n\ngo 1.24\n")
	write(t, filepath.Join(outside, "ext.css"), "@layer components { .ext-class { color: red; } }\n")
	write(t, filepath.Join(root, "go.mod"),
		"module example.com/app\n\ngo 1.24\n\nrequire example.com/ext v0.0.0\n\nreplace example.com/ext => "+outside+"\n")
	sheet := filepath.Join(root, "app.css")
	write(t, sheet, "@layer components { .card { color: red; } }\n")
	write(t, filepath.Join(root, "index.hml"), ".card\n")

	got, err := run(root, []string{sheet}, []string{"example.com/ext"}, nil, nil)
	is.HasErr(err)
	is.Eq(len(got), 0)

	got, err = run(root, []string{sheet}, []string{"example.com/ext"}, []string{outside}, nil)
	is.NoErr(err)
	is.Eq(got, []string{"unused: .ext-class"})
}

func TestModuleDirAcceptsModuleCache(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	dir, err := moduleDir(".", "github.com/croaky/is", nil)
	is.NoErr(err)
	is.True(dir != "")
}

func TestRefsRejectsGoItCannotParse(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()

	// A file the parser cannot read contributes no classes, and silence
	// there reads as "every selector that file named is dead".
	write(t, filepath.Join(root, "web", "handler.go"), "package web\n\nfunc f() {\n")

	_, err := Refs(root, nil)
	is.HasErr(err)
}

// run reads each sheet, reports each class that nothing renders, and
// lints each sheet under its own path. A __ class in the markup fails
// once, whatever the number of sheets.
func TestRunReportsUnusedClassesAndLintsEachSheet(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	root := t.TempDir()

	app := filepath.Join(root, "ui", "css", "app.css")
	site := filepath.Join(root, "site", "site.css")
	write(t, app, `@layer components {
  .card { color: red; }
  .paperclip { color: blue; }
  .tag-priority { color: green; }
}
`)
	write(t, site, `@layer components {
  @media (max-width: 48em) {
    .card { padding: 0; }
  }
}
`)
	write(t, filepath.Join(root, "ui", "views", "a.hml"), ".card\n.row__cell\n")
	write(t, filepath.Join(root, "web", "handler.go"),
		"package web\n\nfunc statusClass() string { return \"tag-priority\" }\n")

	got, err := run(root, []string{app, site}, nil, nil, nil)
	is.NoErr(err)
	is.Eq(got, []string{
		"unused: .paperclip",
		"lint: reference: .row__cell: a referenced class has no __",
		"lint: " + site + ": media: @media (max-width: 48em): write the width as a range, such as (width < 70em)",
	})
}

func TestReadSheetRejectsASheetWithNoClasses(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "app.css")

	write(t, path, "body { margin: 0; }\n")
	r, err := openRoot(dir)
	is.NoErr(err)
	defer r.Close()

	_, err = readSheet(r, "app.css", path)
	is.HasErr(err)
}

// A module's classes come from each of its .css files, and a module can
// have none.
func TestModuleClasses(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	dir := t.TempDir()

	write(t, filepath.Join(dir, "lib.css"), "@layer components { .lib-tip { color: red; } }\n")
	write(t, filepath.Join(dir, "testdata", "fixture.css"), ".fixture { color: red; }\n")

	r, err := openRoot(dir)
	is.NoErr(err)
	defer r.Close()
	got, err := moduleClasses(r)
	is.NoErr(err)
	is.Eq(got, []string{"lib-tip"})

	empty, err := openRoot(t.TempDir())
	is.NoErr(err)
	defer empty.Close()
	got, err = moduleClasses(empty)
	is.NoErr(err)
	is.Eq(len(got), 0)
}

// A module with no sheet counts when its Go names a class, as a package
// that writes classes and leaves the colors to the app does. A module
// that neither defines nor names a class is an error, because the flag
// then names the wrong module.
func TestReadModule(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	writer := t.TempDir()
	write(t, filepath.Join(writer, "html.go"), "package hl\n\nconst keywordClass = \"k\"\n")
	classes, refs, err := readModule(writer, nil)
	is.NoErr(err)
	is.Eq(len(classes), 0)
	is.True(refs["k"])

	empty := t.TempDir()
	write(t, filepath.Join(empty, "doc.go"), "package empty\n")
	_, _, err = readModule(empty, nil)
	is.HasErr(err)
}
