// Command depbudget fails when go.mod has more direct dependencies than the
// budget of docs/security-baseline.md §8. Modules required only by a tool
// directive do not count: go mod tidy marks them indirect, because no
// package of the module imports them.
//
// Usage, from the repository root:
//
//	go run ./tools/depbudget -max 6
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	exitOverBudget = 1
	exitUsage      = 2
)

// goMod is the part of `go mod edit -json` that the budget needs.
type goMod struct {
	Require []struct {
		Path     string
		Indirect bool
	}
}

func main() {
	maxDirect := flag.Int("max", 0, "maximum number of direct dependencies (required, > 0)")
	flag.Parse()
	if *maxDirect <= 0 || flag.NArg() > 0 {
		flag.Usage()
		os.Exit(exitUsage)
	}
	mod, err := readGoMod(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "depbudget:", err)
		os.Exit(exitUsage)
	}
	direct := directDeps(mod)
	if len(direct) > *maxDirect {
		fmt.Fprintf(os.Stderr, "depbudget: %d direct dependencies, budget is %d:\n", len(direct), *maxDirect)
		for _, p := range direct {
			fmt.Fprintln(os.Stderr, "  "+p)
		}
		os.Exit(exitOverBudget)
	}
	fmt.Printf("depbudget: ok, %d of %d direct dependencies\n", len(direct), *maxDirect)
}

// readGoMod asks the go command for the parsed go.mod, so the syntax rules
// (blocks, comments, quoting) are Go's own.
func readGoMod(ctx context.Context) (goMod, error) {
	out, err := exec.CommandContext(ctx, "go", "mod", "edit", "-json").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return goMod{}, fmt.Errorf("go mod edit -json: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return goMod{}, fmt.Errorf("go mod edit -json: %w", err)
	}
	var mod goMod
	if err := json.Unmarshal(out, &mod); err != nil {
		return goMod{}, fmt.Errorf("decode go mod edit -json: %w", err)
	}
	return mod, nil
}

func directDeps(mod goMod) []string {
	var direct []string
	for _, r := range mod.Require {
		if !r.Indirect {
			direct = append(direct, r.Path)
		}
	}
	return direct
}
