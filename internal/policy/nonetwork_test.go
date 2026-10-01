package policy

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// banned reports whether an import path could make network connections.
func banned(path string) bool {
	switch {
	case path == "net", strings.HasPrefix(path, "net/"):
		return true
	case path == "crypto/tls":
		return true
	case strings.HasPrefix(path, "golang.org/x/net"):
		return true
	}
	return false
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestNoNetworkImports fails if any Go file in the repository (including
// tests) imports a network package.
func TestNoNetworkImports(t *testing.T) {
	root := repoRoot(t)
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "dist" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if banned(p) {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s imports %q; the scanner must make no network connections", rel, p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// neverAllowed are packages that make web or secure connections. They must
// not be compiled into the scanner at all.
func neverAllowed(path string) bool {
	switch path {
	case "net/http", "crypto/tls", "net/rpc", "net/smtp", "net/mail":
		return true
	}
	return strings.HasPrefix(path, "net/http/") || strings.HasPrefix(path, "golang.org/x/net")
}

// allowedNetImporters may import "net" (see doc.go for why).
var allowedNetImporters = map[string]bool{
	"golang.org/x/sys/windows": true,
}

// TestNoNetworkDependencies checks every package compiled into the Windows
// scanner, including the standard library and golang.org/x/sys, so that a
// network package cannot sneak in through a dependency.
func TestNoNetworkDependencies(t *testing.T) {
	gobin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go command not available")
	}
	for _, arch := range []string{"amd64", "arm64"} {
		cmd := exec.Command(gobin, "list", "-deps", "-f", "{{.ImportPath}} {{.Standard}} {{join .Imports \",\"}}", "./cmd/scanner")
		cmd.Dir = repoRoot(t)
		cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH="+arch, "CGO_ENABLED=0")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go list failed: %v\n%s", err, out)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			pkg, standard := fields[0], fields[1] == "true"
			if neverAllowed(pkg) {
				t.Errorf("windows/%s scanner includes %q", arch, pkg)
			}
			if standard || len(fields) < 3 {
				continue // the standard library's own wiring (net -> net/netip) is fine
			}
			for _, imp := range strings.Split(fields[2], ",") {
				if banned(imp) && !allowedNetImporters[pkg] {
					t.Errorf("windows/%s: %s imports %q", arch, pkg, imp)
				}
			}
		}
	}
}
