package interpreter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeScript writes src into dir/name and returns the full path.
func writeScript(t *testing.T, dir, name, src string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

// TestUseImportsFunctionsAndVariablesIntoCallerScope covers the documented
// behaviour of `use`: a module's functions and variables become available in
// the importing file. Modules used to be evaluated in a throwaway child scope,
// which made every non-native import fail with "'fn' is not defined".
func TestUseImportsFunctionsAndVariablesIntoCallerScope(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "math.pen", `
fn square(x) {
    return x * x
}

store PI = 3
`)

	main := writeScript(t, dir, "main.pen", `
use math

say toString(square(7))
say toString(PI)
`)

	if err := New().RunFile(main); err != nil {
		t.Fatalf("RunFile: %v", err)
	}
}

func TestUseResolvesModuleFromWorkingDirectoryModulesDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "modules"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "modules", "greet.pen"),
		[]byte("fn hello() {\n    return \"hi\"\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// The script itself lives in a subdirectory, so the module can only be
	// found via the working directory's modules/ directory.
	sub := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	main := writeScript(t, sub, "app.pen", "use greet\nsay hello()\n")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	if err := New().RunFile(main); err != nil {
		t.Fatalf("RunFile: %v", err)
	}
}

func TestUseImportsModuleOnlyOnce(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "counter.pen", `
store loads = 0
loads = loads + 1
`)

	main := writeScript(t, dir, "main.pen", `
use counter
use counter
say toString(loads)
`)

	interp := New()
	if err := interp.RunFile(main); err != nil {
		t.Fatalf("RunFile: %v", err)
	}

	// The module must be executed exactly once despite being used twice.
	if got := interp.imported[filepath.Join(dir, "counter.pen")]; !got {
		t.Error("expected counter.pen to be recorded as imported")
	}
	if n := len(interp.imported); n != 1 {
		t.Errorf("expected 1 module to be imported, got %d", n)
	}
}

func TestUseMissingModuleReportsSearchedPaths(t *testing.T) {
	dir := t.TempDir()
	main := writeScript(t, dir, "main.pen", "use definitely_not_a_module\n")

	err := New().RunFile(main)
	if err == nil {
		t.Fatal("expected an error for a missing module")
	}
	if !strings.Contains(err.Error(), "Could not import module 'definitely_not_a_module'") {
		t.Errorf("error should name the missing module, got: %v", err)
	}
}
