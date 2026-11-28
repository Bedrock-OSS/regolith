package regolith

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestWatchConfig(t *testing.T, watchPaths ...string) (*Config, string, string, string) {
	t.Helper()
	root := t.TempDir()
	bpDir := filepath.Join(root, "packs", "BP")
	rpDir := filepath.Join(root, "packs", "RP")
	dataDir := filepath.Join(root, "packs", "data")
	for _, dir := range []string{bpDir, rpDir, dataDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}
	cfg := &Config{
		Name:   "watch_test",
		Author: "test",
		Packs: Packs{
			BehaviorFolder: bpDir,
			ResourceFolder: rpDir,
		},
		RegolithProject: RegolithProject{
			DataPath:   dataDir,
			WatchPaths: watchPaths,
		},
	}
	return cfg, bpDir, rpDir, dataDir
}

func startTestWatcher(t *testing.T, cfg *Config) (*DirWatcher, chan string, chan error) {
	t.Helper()
	interruption := make(chan string)
	errCh := make(chan error)
	stage := make(chan string)
	w, err := newDirWatcher(cfg, interruption, errCh, stage)
	if err != nil {
		t.Fatalf("newDirWatcher failed: %v", err)
	}
	t.Cleanup(func() {
		_ = w.Close()
	})
	return w, interruption, errCh
}

func expectInterruption(t *testing.T, interruption <-chan string, errCh <-chan error, want string, timeout time.Duration) {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case got := <-interruption:
		if got != want {
			t.Fatalf("expected interruption %q, got %q", want, got)
		}
	case err := <-errCh:
		t.Fatalf("unexpected watcher error while waiting for %q: %v", want, err)
	case <-timer.C:
		t.Fatalf("timed out waiting for interruption %q", want)
	}
}

func expectNoInterruption(t *testing.T, interruption <-chan string, errCh <-chan error, duration time.Duration) {
	t.Helper()
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case got := <-interruption:
		t.Fatalf("expected no interruption, got %q", got)
	case err := <-errCh:
		t.Fatalf("unexpected watcher error: %v", err)
	case <-timer.C:
	}
}

func drainInterruptions(interruption <-chan string, errCh <-chan error, t *testing.T) {
	t.Helper()
	for {
		select {
		case <-interruption:
		case err := <-errCh:
			t.Fatalf("unexpected watcher error while draining: %v", err)
		default:
			return
		}
	}
}

func waitPastDebounce() {
	time.Sleep(watchDebounceDuration + 60*time.Millisecond)
}

func TestDirWatcherDetectsChangesInRoots(t *testing.T) {
	extraDir := filepath.Join(t.TempDir(), "extra")
	if err := os.MkdirAll(extraDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfg, bpDir, rpDir, dataDir := newTestWatchConfig(t, extraDir)
	_, interruption, errCh := startTestWatcher(t, cfg)

	cases := []struct {
		dir    string
		source string
	}{
		{bpDir, "bp"},
		{rpDir, "rp"},
		{dataDir, "data"},
		{extraDir, "extras"},
	}

	for _, tc := range cases {
		filePath := filepath.Join(tc.dir, "change.txt")
		if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filePath, err)
		}
		expectInterruption(t, interruption, errCh, tc.source, 3*time.Second)
		waitPastDebounce()
		drainInterruptions(interruption, errCh, t)
	}
}

func TestDirWatcherNestedCreateAndDelete(t *testing.T) {
	cfg, bpDir, _, _ := newTestWatchConfig(t)
	_, interruption, errCh := startTestWatcher(t, cfg)

	nestedDir := filepath.Join(bpDir, "1", "2", "3")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	nestedFile := filepath.Join(nestedDir, "4.txt")
	if err := os.WriteFile(nestedFile, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}
	expectInterruption(t, interruption, errCh, "bp", 3*time.Second)
	waitPastDebounce()
	drainInterruptions(interruption, errCh, t)

	if err := os.WriteFile(nestedFile, []byte("updated content"), 0644); err != nil {
		t.Fatal(err)
	}
	expectInterruption(t, interruption, errCh, "bp", 3*time.Second)
	waitPastDebounce()
	drainInterruptions(interruption, errCh, t)

	if err := os.RemoveAll(filepath.Join(bpDir, "1")); err != nil {
		t.Fatalf("failed to remove nested directory while watching: %v", err)
	}
	expectInterruption(t, interruption, errCh, "bp", 3*time.Second)
}

func TestDirWatcherPauseAndRestart(t *testing.T) {
	cfg, bpDir, _, dataDir := newTestWatchConfig(t)
	w, interruption, errCh := startTestWatcher(t, cfg)

	subData := filepath.Join(dataDir, "run_counter")
	if err := os.MkdirAll(subData, 0755); err != nil {
		t.Fatal(err)
	}
	expectInterruption(t, interruption, errCh, "data", 3*time.Second)
	waitPastDebounce()
	drainInterruptions(interruption, errCh, t)

	if err := w.Pause(); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}

	if err := os.RemoveAll(subData); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(subData, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subData, "counter.txt"), []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := w.Restart(); err != nil {
		t.Fatalf("Restart failed: %v", err)
	}

	expectNoInterruption(t, interruption, errCh, 250*time.Millisecond)

	if err := os.WriteFile(filepath.Join(bpDir, "after_restart.txt"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	expectInterruption(t, interruption, errCh, "bp", 3*time.Second)
}

func TestDirWatcherPauseDoesNotDeadlockWithUnreadInterruption(t *testing.T) {
	cfg, bpDir, _, _ := newTestWatchConfig(t)
	w, interruption, errCh := startTestWatcher(t, cfg)

	if err := os.WriteFile(filepath.Join(bpDir, "unread.txt"), []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)

	pauseDone := make(chan error, 1)
	go func() {
		pauseDone <- w.Pause()
	}()
	select {
	case err := <-pauseDone:
		if err != nil {
			t.Fatalf("Pause failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Pause deadlocked while an interruption was unread")
	}

	restartDone := make(chan error, 1)
	go func() {
		restartDone <- w.Restart()
	}()
	select {
	case err := <-restartDone:
		if err != nil {
			t.Fatalf("Restart failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Restart deadlocked")
	}

	expectInterruption(t, interruption, errCh, "bp", 3*time.Second)
}

func TestDirWatcherRapidCreateDeleteStress(t *testing.T) {
	cfg, bpDir, _, _ := newTestWatchConfig(t)
	_, interruption, errCh := startTestWatcher(t, cfg)

	for i := range 20 {
		dir := filepath.Join(bpDir, fmt.Sprintf("stress_%d", i), "nested")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(dir, "item.txt")
		if err := os.WriteFile(file, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(filepath.Join(bpDir, fmt.Sprintf("stress_%d", i))); err != nil {
			t.Fatal(err)
		}
		drainInterruptions(interruption, errCh, t)
	}

	waitPastDebounce()
	drainInterruptions(interruption, errCh, t)

	if err := os.WriteFile(filepath.Join(bpDir, "after_stress.txt"), []byte("still alive"), 0644); err != nil {
		t.Fatal(err)
	}
	expectInterruption(t, interruption, errCh, "bp", 3*time.Second)
}

func TestDirWatcherSingleFileWatchPath(t *testing.T) {
	extraDir := filepath.Join(t.TempDir(), "files")
	if err := os.MkdirAll(extraDir, 0755); err != nil {
		t.Fatal(err)
	}
	watchedFile := filepath.Join(extraDir, "watched.json")
	siblingFile := filepath.Join(extraDir, "sibling.json")
	if err := os.WriteFile(watchedFile, []byte(`{"v":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(siblingFile, []byte(`{"s":1}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, _, _ := newTestWatchConfig(t, watchedFile)
	_, interruption, errCh := startTestWatcher(t, cfg)

	if err := os.WriteFile(siblingFile, []byte(`{"s":222}`), 0644); err != nil {
		t.Fatal(err)
	}
	expectNoInterruption(t, interruption, errCh, 250*time.Millisecond)

	if err := os.WriteFile(watchedFile, []byte(`{"v":22222}`), 0644); err != nil {
		t.Fatal(err)
	}
	expectInterruption(t, interruption, errCh, "extras", 3*time.Second)
}

func TestIsInDir(t *testing.T) {
	base := t.TempDir()
	rpDir := filepath.Join(base, "RP")
	if err := os.MkdirAll(rpDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		root string
		want bool
	}{
		{name: "empty root", path: filepath.Join(rpDir, "a.json"), root: "", want: false},
		{name: "exact root", path: rpDir, root: rpDir, want: true},
		{name: "nested file", path: filepath.Join(rpDir, "sub", "a.json"), root: rpDir, want: true},
		{name: "dotdot prefixed filename inside dir", path: filepath.Join(rpDir, "..foo.json"), root: rpDir, want: true},
		{name: "sibling with same prefix", path: filepath.Join(base, "RP_other", "a.json"), root: rpDir, want: false},
		{name: "parent directory", path: base, root: rpDir, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isInDir(tc.path, tc.root); got != tc.want {
				t.Fatalf("isInDir(%q, %q) = %v, want %v", tc.path, tc.root, got, tc.want)
			}
		})
	}
}
