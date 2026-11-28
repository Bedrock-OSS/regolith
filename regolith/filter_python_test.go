package regolith

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func createPythonFilterFixture(t *testing.T, requirements string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.py"), []byte("print('hello')\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(requirements), 0644); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "main.py")
}

func buildFakeFailingVenvPython(t *testing.T) string {
	t.Helper()
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module fakevenv\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mainGo := `package main

import (
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) >= 4 && os.Args[1] == "-m" && os.Args[2] == "venv" {
		venvDir := os.Args[3]
		_ = os.MkdirAll(venvDir, 0755)
		_ = os.WriteFile(filepath.Join(venvDir, "partial.txt"), []byte("partial"), 0644)
	}
	os.Exit(1)
}
`
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(t.TempDir(), "fake-python"+exeSuffix)
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = srcDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build fake python binary: %v\n%s", err, out)
	}
	return binPath
}

func TestPythonInstallDependencies_MissingPythonDoesNotCreateVenv(t *testing.T) {
	isolateTestCache(t)

	missingPython := filepath.Join(t.TempDir(), "nonexistent-python"+exeSuffix)
	cfg := NewUserConfig()
	cfg.fillDefaults()
	cfg.PythonRunner = &missingPython
	setTestUserConfig(t, cfg)

	dotRegolithPath := filepath.Join(t.TempDir(), ".regolith")
	scriptPath := createPythonFilterFixture(t, "")
	def := &PythonFilterDefinition{
		FilterDefinition: FilterDefinition{Id: "py-missing"},
		Script:           scriptPath,
		VenvSlot:         0,
	}

	err := def.InstallDependencies(nil, dotRegolithPath)
	if err == nil {
		t.Fatal("expected InstallDependencies to fail when Python is missing")
	}
	if !strings.Contains(err.Error(), "Python not found") {
		t.Fatalf("expected Python not found error, got: %v", err)
	}
	venvPath := filepath.Join(dotRegolithPath, "cache", "venvs", "0")
	if _, statErr := os.Stat(venvPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected no venv directory to be created at %s, statErr=%v", venvPath, statErr)
	}
}

func TestPythonInstallDependencies_CleansUpPartialVenvOnPipFailure(t *testing.T) {
	isolateTestCache(t)
	setOfflinePipEnv(t)

	cfg := NewUserConfig()
	cfg.fillDefaults()
	setTestUserConfig(t, cfg)

	if _, err := findPython(); err != nil {
		t.Skipf("python not available: %v", err)
	}

	t.Run("new venv removed on pip failure", func(t *testing.T) {
		dotRegolithPath := filepath.Join(t.TempDir(), ".regolith")
		scriptPath := createPythonFilterFixture(t, "nonexistent-regolith-test-pkg==0.0.0\n")
		def := &PythonFilterDefinition{
			FilterDefinition: FilterDefinition{Id: "py-pip-fail"},
			Script:           scriptPath,
			VenvSlot:         1,
		}

		err := def.InstallDependencies(nil, dotRegolithPath)
		if err == nil {
			t.Fatal("expected InstallDependencies to fail for nonexistent pip package")
		}
		venvPath := filepath.Join(dotRegolithPath, "cache", "venvs", "1")
		if _, statErr := os.Stat(venvPath); !os.IsNotExist(statErr) {
			t.Fatalf("expected partial venv %s to be removed, statErr=%v", venvPath, statErr)
		}
	})

	t.Run("pre-existing venv preserved on pip failure", func(t *testing.T) {
		dotRegolithPath := filepath.Join(t.TempDir(), ".regolith")
		okScript := createPythonFilterFixture(t, "")
		okDef := &PythonFilterDefinition{
			FilterDefinition: FilterDefinition{Id: "py-ok"},
			Script:           okScript,
			VenvSlot:         2,
		}
		if err := okDef.InstallDependencies(nil, dotRegolithPath); err != nil {
			t.Fatalf("expected initial venv creation to succeed, got: %v", err)
		}

		venvPath := filepath.Join(dotRegolithPath, "cache", "venvs", "2")
		sentinel := filepath.Join(venvPath, "sentinel.txt")
		if err := os.WriteFile(sentinel, []byte("keep"), 0644); err != nil {
			t.Fatal(err)
		}

		failScript := createPythonFilterFixture(t, "nonexistent-regolith-test-pkg==0.0.0\n")
		failDef := &PythonFilterDefinition{
			FilterDefinition: FilterDefinition{Id: "py-pip-fail-shared"},
			Script:           failScript,
			VenvSlot:         2,
		}
		if err := failDef.InstallDependencies(nil, dotRegolithPath); err == nil {
			t.Fatal("expected InstallDependencies to fail for nonexistent pip package")
		}
		if _, err := os.Stat(sentinel); err != nil {
			t.Fatalf("expected pre-existing venv sentinel %s to be preserved: %v", sentinel, err)
		}
		venvPython := filepath.Join(venvPath, venvScriptsPath, "python"+exeSuffix)
		if _, err := os.Stat(venvPython); err != nil {
			t.Fatalf("expected pre-existing venv python %s to be preserved: %v", venvPython, err)
		}
	})
}

func TestPythonInstallDependencies_CleansUpPartialVenvOnVenvCreationFailure(t *testing.T) {
	isolateTestCache(t)

	fakePython := buildFakeFailingVenvPython(t)
	cfg := NewUserConfig()
	cfg.fillDefaults()
	cfg.PythonRunner = &fakePython
	setTestUserConfig(t, cfg)

	t.Run("new venv removed on venv creation failure", func(t *testing.T) {
		dotRegolithPath := filepath.Join(t.TempDir(), ".regolith")
		scriptPath := createPythonFilterFixture(t, "")
		def := &PythonFilterDefinition{
			FilterDefinition: FilterDefinition{Id: "py-venv-fail"},
			Script:           scriptPath,
			VenvSlot:         3,
		}

		err := def.InstallDependencies(nil, dotRegolithPath)
		if err == nil {
			t.Fatal("expected InstallDependencies to fail when python -m venv fails")
		}
		venvPath := filepath.Join(dotRegolithPath, "cache", "venvs", "3")
		if _, statErr := os.Stat(venvPath); !os.IsNotExist(statErr) {
			t.Fatalf("expected partial venv %s to be removed, statErr=%v", venvPath, statErr)
		}
	})

	t.Run("pre-existing venv preserved on venv creation failure", func(t *testing.T) {
		dotRegolithPath := filepath.Join(t.TempDir(), ".regolith")
		venvPath := filepath.Join(dotRegolithPath, "cache", "venvs", "4")
		if err := os.MkdirAll(venvPath, 0755); err != nil {
			t.Fatal(err)
		}
		sentinel := filepath.Join(venvPath, "sentinel.txt")
		if err := os.WriteFile(sentinel, []byte("keep"), 0644); err != nil {
			t.Fatal(err)
		}

		scriptPath := createPythonFilterFixture(t, "")
		def := &PythonFilterDefinition{
			FilterDefinition: FilterDefinition{Id: "py-venv-fail-shared"},
			Script:           scriptPath,
			VenvSlot:         4,
		}
		if err := def.InstallDependencies(nil, dotRegolithPath); err == nil {
			t.Fatal("expected InstallDependencies to fail when python -m venv fails")
		}
		if _, err := os.Stat(sentinel); err != nil {
			t.Fatalf("expected pre-existing venv sentinel %s to be preserved: %v", sentinel, err)
		}
	})
}
