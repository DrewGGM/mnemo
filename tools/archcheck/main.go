// Command archcheck fails when a package imports across the module
// boundaries of docs/architecture.md §5 (ADR-0007, decision 3). Test files
// are checked too, whatever their build tags.
//
// Usage, from the repository root:
//
//	go run ./tools/archcheck
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"
	"testing/fstest"
)

// pureCores must not reach storage, model providers or the network: they
// take everything as arguments and return decisions.
var pureCores = []string{"internal/resolve", "internal/retrieve", "internal/policy"}

// Each path also forbids its subpackages.
var (
	forbiddenModulePkgsInCores = []string{"internal/store", "internal/provider"}
	forbiddenStdPkgsInCores    = []string{"net/http", "database/sql"}
)

// adapters translate a protocol into calls on the root package only.
var adapters = []string{"internal/httpapi", "internal/mcpserver"}

const (
	allowedInAdapters      = "internal/config"
	allowedInAdapterTests  = "internal/store/storetest"
	selfCheckModule        = "example.com/m"
	exitViolations         = 1
	exitCheckerUnavailable = 2
)

// violation is one forbidden import, reported as file and import path.
type violation struct {
	file, imp string
}

func main() {
	if err := selfCheck(); err != nil {
		fmt.Fprintln(os.Stderr, "archcheck: self-check failed:", err)
		os.Exit(exitCheckerUnavailable)
	}
	module, err := modulePath(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "archcheck:", err)
		os.Exit(exitCheckerUnavailable)
	}
	vs, err := check(os.DirFS("."), module)
	if err != nil {
		fmt.Fprintln(os.Stderr, "archcheck:", err)
		os.Exit(exitCheckerUnavailable)
	}
	for _, v := range vs {
		fmt.Fprintf(os.Stderr, "%s: forbidden import %q\n", v.file, v.imp)
	}
	if len(vs) > 0 {
		os.Exit(exitViolations)
	}
	fmt.Println("archcheck: ok")
}

// modulePath asks the go command for the module path, so the syntax rules
// of go.mod (comments, quoting) are Go's own.
func modulePath(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "go", "mod", "edit", "-json").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("go mod edit -json: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("go mod edit -json: %w", err)
	}
	var mod struct{ Module struct{ Path string } }
	if err := json.Unmarshal(out, &mod); err != nil {
		return "", fmt.Errorf("decode go mod edit -json: %w", err)
	}
	if mod.Module.Path == "" {
		return "", errors.New("go.mod has no module directive")
	}
	return mod.Module.Path, nil
}

// check walks every Go file under the root of fsys, the way the go command
// sees packages: directories named testdata or starting with "." or "_"
// are skipped.
func check(fsys fs.FS, module string) ([]violation, error) {
	var vs []violation
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name != "." && skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		found, err := checkFile(fsys, name, module)
		vs = append(vs, found...)
		return err
	})
	return vs, err
}

func skipDir(base string) bool {
	return base == "testdata" || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_")
}

func checkFile(fsys fs.FS, name, module string) ([]violation, error) {
	src, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	dir := path.Dir(name)
	isTest := strings.HasSuffix(name, "_test.go")
	var vs []violation
	for _, spec := range f.Imports {
		imp, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if forbidden(dir, imp, module, isTest) {
			vs = append(vs, violation{file: name, imp: imp})
		}
	}
	return vs, nil
}

// forbidden applies the rule of the package in dir to one import.
func forbidden(dir, imp, module string, isTest bool) bool {
	rel, inModule := strings.CutPrefix(imp, module+"/")
	switch {
	case within(dir, pureCores):
		return within(imp, forbiddenStdPkgsInCores) || inModule && within(rel, forbiddenModulePkgsInCores)
	case within(dir, adapters):
		return inModule && under(rel, "internal") && !allowedInAdapter(dir, rel, isTest)
	}
	return false
}

// allowedInAdapter takes a module-relative internal path. An adapter's own
// tree is allowed because an external test package imports the package it tests.
func allowedInAdapter(dir, rel string, isTest bool) bool {
	return under(rel, dir) || rel == allowedInAdapters || isTest && rel == allowedInAdapterTests
}

// within reports whether p is under any of roots.
func within(p string, roots []string) bool {
	return slices.ContainsFunc(roots, func(root string) bool { return under(p, root) })
}

// under reports whether p is root or one of its subpaths.
func under(p, root string) bool {
	return p == root || strings.HasPrefix(p, root+"/")
}

// selfCheck runs the checker on a fixture with known violations, so a broken
// rule cannot turn into a silent pass. The fixture covers a forbidden import
// that appears only in a _test.go file.
func selfCheck() error {
	fixture := fstest.MapFS{
		"internal/resolve/plan.go":           goFile("resolve", "example.com/m/internal/store"),
		"internal/retrieve/fuse_test.go":     goFile("retrieve", "net/http/httptest"),
		"internal/policy/policy.go":          goFile("policy", "time"),
		"internal/policy/policy_test.go":     goFile("policy", "database/sql"),
		"internal/httpapi/server.go":         goFile("httpapi", "example.com/m", "example.com/m/internal/config"),
		"internal/httpapi/server_test.go":    goFile("httpapi", "example.com/m/internal/store"),
		"internal/httpapi/ok_test.go":        goFile("httpapi_test", "example.com/m/internal/httpapi", "example.com/m/internal/store/storetest"),
		"internal/mcpserver/server.go":       goFile("mcpserver", "example.com/m/internal/store/storetest"),
		"internal/store/store.go":            goFile("store", "database/sql"),
		"internal/httpapi/testdata/x/bad.go": goFile("x", "example.com/m/internal/store"),
	}
	want := []violation{
		{"internal/httpapi/server_test.go", "example.com/m/internal/store"},
		{"internal/mcpserver/server.go", "example.com/m/internal/store/storetest"},
		{"internal/policy/policy_test.go", "database/sql"},
		{"internal/resolve/plan.go", "example.com/m/internal/store"},
		{"internal/retrieve/fuse_test.go", "net/http/httptest"},
	}
	got, err := check(fixture, selfCheckModule)
	if err != nil {
		return err
	}
	if !slices.Equal(got, want) {
		return fmt.Errorf("got violations %v, want %v", got, want)
	}
	return nil
}

func goFile(pkg string, imports ...string) *fstest.MapFile {
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	for _, imp := range imports {
		fmt.Fprintf(&b, "import _ %q\n", imp)
	}
	return &fstest.MapFile{Data: []byte(b.String())}
}
