package cleanup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/testutil"
	"github.com/gin31259461/homebase/internal/ui"
)

func TestCleanupItemsExposeDefaultsAndOrphanState(t *testing.T) {
	r := &testutil.Runner{
		Outputs: map[string]string{
			"pacman -Qdtq":                   "old-lib\nunused-tool\n",
			"pacman -Qi old-lib unused-tool": "Name            : old-lib\nInstalled Size  : 2.0 MiB\n\nName            : unused-tool\nInstalled Size  : 512.0 KiB\n",
		},
	}
	tasks := []config.CleanupTask{
		{Key: "orphans", Label: "Orphans", Detail: "pacman -Rns", Default: true, Sudo: true},
	}
	items := CleanupItems(r, tasks)
	if len(items) != 1 {
		t.Fatalf("items = %#v; want one item", items)
	}
	if !items[0].DefaultSelected {
		t.Fatal("default cleanup task was not preselected")
	}
	if items[0].State != ui.SelectStateBad {
		t.Fatalf("state = %s; want bad", items[0].State)
	}
	if items[0].DetailValue != "2 orphaned package(s), 2.5 MiB" {
		t.Fatalf("detail value = %q; want orphan summary", items[0].DetailValue)
	}
	if items[0].Detail != "pacman -Rns" {
		t.Fatalf("detail = %q; want cleanup command detail", items[0].Detail)
	}
	wantInspectPrefix := "Label: Orphans\nCommand: pacman -Rns\nSudo: true\nReview before removal.\nKeep a package: sudo pacman -D --asexplicit <package>\nRemoval uses pacman confirmation; --noconfirm is not used.\nPackage count: 2\nInstalled size: 2.5 MiB\nPackages:"
	if !strings.HasPrefix(items[0].Inspect, wantInspectPrefix) {
		t.Fatalf("inspect = %q; want size/count before package list", items[0].Inspect)
	}
}

func TestOrphanRemovalArgsUsePacmanConfirmation(t *testing.T) {
	got := orphanRemovalArgs([]string{"old-lib", "unused-tool"})
	want := []string{"pacman", "-Rns", "old-lib", "unused-tool"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("args = %#v; want %#v", got, want)
	}
	for _, arg := range got {
		if arg == "--noconfirm" {
			t.Fatalf("orphan removal args must not include --noconfirm: %#v", got)
		}
	}
}

func TestOrphanCleanupTreatsEmptyPacmanFailureAsNoOrphans(t *testing.T) {
	r := &testutil.Runner{
		Errors: map[string]error{
			"pacman -Qdtq": testutil.Err(),
		},
	}
	info := orphanCleanupInfo(r)
	if info.state != ui.SelectStateGood {
		t.Fatalf("state = %s; want good", info.state)
	}
	if info.summary != "no orphaned packages" {
		t.Fatalf("summary = %q; want no orphaned packages", info.summary)
	}

	pkgs, err := orphanPackages(r)
	if err != nil {
		t.Fatalf("orphanPackages returned error for empty pacman output: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("pkgs = %#v; want none", pkgs)
	}
}

func TestDirCleanupInfoStates(t *testing.T) {
	dir := t.TempDir()
	r := &testutil.Runner{}
	info := dirCleanupInfo(r, dir, "cache")
	if info.state != ui.SelectStateGood {
		t.Fatalf("empty state = %s; want good", info.state)
	}
	if info.summary != "0 B" {
		t.Fatalf("empty summary = %q; want 0 B", info.summary)
	}
	if err := os.WriteFile(filepath.Join(dir, "cache.bin"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	info = dirCleanupInfo(r, dir, "cache")
	if info.state != ui.SelectStatePartial {
		t.Fatalf("small state = %s; want partial", info.state)
	}
	if info.summary != "4 B" {
		t.Fatalf("small summary = %q; want 4 B", info.summary)
	}
}

func TestCleanupSizeStateThreshold(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  ui.SelectState
	}{
		{name: "empty", bytes: 0, want: ui.SelectStateGood},
		{name: "small", bytes: SmallCleanupBytes - 1, want: ui.SelectStatePartial},
		{name: "large", bytes: SmallCleanupBytes, want: ui.SelectStateBad},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanupSizeState(tt.bytes); got != tt.want {
				t.Fatalf("CleanupSizeState(%d) = %s; want %s", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestDirCleanupInfoUsesDuSize(t *testing.T) {
	r := &testutil.Runner{
		Outputs: map[string]string{
			"du -sb /var/cache/pacman/pkg": "10485760\t/var/cache/pacman/pkg\n",
		},
	}
	info := dirCleanupInfo(r, "/var/cache/pacman/pkg", "Pacman package cache")
	if info.summary != "10.0 MiB" {
		t.Fatalf("summary = %q; want 10.0 MiB", info.summary)
	}
}

func TestDirCleanupInfoUsesDuOutputWhenDuExitsNonZero(t *testing.T) {
	r := &duErrorRunner{output: "6817010741\t/var/cache/pacman/pkg\n"}
	info := dirCleanupInfo(r, "/var/cache/pacman/pkg", "Pacman package cache")
	if info.summary != "6.3 GiB" {
		t.Fatalf("summary = %q; want 6.3 GiB", info.summary)
	}
}

func TestPacmanCacheCleanupInfoShowsReclaimableSize(t *testing.T) {
	dir := t.TempDir()
	oldPkg := filepath.Join(dir, "old.pkg.tar.zst")
	if err := os.WriteFile(oldPkg, make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &testutil.Runner{
		Outputs: map[string]string{
			"du -sb " + dir:          "10485760\t" + dir + "\n",
			"paccache -dq -c " + dir: oldPkg + "\n",
		},
	}
	info := pacmanCacheCleanupInfo(r, dir)
	if info.summary != "2.0 KiB reclaimable" {
		t.Fatalf("summary = %q; want reclaimable size", info.summary)
	}
	if !strings.Contains(strings.Join(info.inspect, "\n"), "Total cache: 10.0 MiB") {
		t.Fatalf("inspect = %#v; want total cache", info.inspect)
	}
}

func TestPacmanCacheCleanupInfoShowsZeroWhenPaccacheHasNoCandidates(t *testing.T) {
	r := &testutil.Runner{
		Outputs: map[string]string{
			"du -sb /var/cache/pacman/pkg":          "6817010741\t/var/cache/pacman/pkg\n",
			"paccache -dq -c /var/cache/pacman/pkg": "",
		},
	}
	info := pacmanCacheCleanupInfo(r, "/var/cache/pacman/pkg")
	if info.state != ui.SelectStateGood {
		t.Fatalf("state = %s; want good", info.state)
	}
	if info.summary != "0 B reclaimable" {
		t.Fatalf("summary = %q; want 0 B reclaimable", info.summary)
	}
}

func TestJournalCleanupInfoShowsReclaimableOverVacuumTarget(t *testing.T) {
	r := &testutil.Runner{
		Outputs: map[string]string{
			"journalctl --disk-usage": "Archived and active journals take up 150.0M in the file system.\n",
		},
	}
	info := journalCleanupInfo(r)
	if info.state != ui.SelectStateBad {
		t.Fatalf("state = %s; want bad", info.state)
	}
	if info.summary != "50.0 MiB over 100.0 MiB target" {
		t.Fatalf("summary = %q; want reclaimable over target", info.summary)
	}
	inspect := strings.Join(info.inspect, "\n")
	if !strings.Contains(inspect, "Total journal usage: 150.0 MiB") || !strings.Contains(inspect, "Vacuum target: 100.0 MiB") {
		t.Fatalf("inspect = %#v; want total and target", info.inspect)
	}
}

func TestJournalCleanupInfoShowsZeroWhenUnderVacuumTarget(t *testing.T) {
	r := &testutil.Runner{
		Outputs: map[string]string{
			"journalctl --disk-usage": "Archived and active journals take up 16.0M in the file system.\n",
		},
	}
	info := journalCleanupInfo(r)
	if info.state != ui.SelectStateGood {
		t.Fatalf("state = %s; want good", info.state)
	}
	if info.summary != "0 B over 100.0 MiB target" {
		t.Fatalf("summary = %q; want zero reclaimable", info.summary)
	}
}

func TestRunJournalTaskUsesVacuumSizeTarget(t *testing.T) {
	r := &testutil.Runner{}
	if err := RunTask(r, "journal"); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(r.Calls, "\n"), "sudo journalctl --vacuum-size=100M"; got != want {
		t.Fatalf("calls = %q; want %q", got, want)
	}
}

func TestNPMCacheCleanupInfoScansActualCachePayload(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "_cacache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "payload"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "_logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_logs", "debug.log"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	r := &testutil.Runner{
		Outputs: map[string]string{
			"npm config get cache": root + "\n",
		},
	}
	info := npmCacheCleanupInfo(r)
	if info.summary != "2.0 KiB" {
		t.Fatalf("summary = %q; want _cacache size only", info.summary)
	}
	inspect := strings.Join(info.inspect, "\n")
	if !strings.Contains(inspect, "npm content-addressable cache: 2.0 KiB") || !strings.Contains(inspect, "npm cache root: "+root) {
		t.Fatalf("inspect = %#v; want _cacache path and root", info.inspect)
	}
}

func TestParseSize(t *testing.T) {
	got, ok := parseSize("Archived and active journals take up 16.0M in the file system.")
	if !ok {
		t.Fatal("size was not parsed")
	}
	if want := int64(16 * 1024 * 1024); got != want {
		t.Fatalf("size = %d; want %d", got, want)
	}
	if _, ok := parseSize("keeps last 3 versions"); ok {
		t.Fatal("bare number should not parse as a size")
	}
}

type duErrorRunner struct {
	testutil.Runner
	output string
}

func (r *duErrorRunner) Capture(name string, args ...string) (string, error) {
	if name == "du" && len(args) == 2 && args[0] == "-sb" {
		return r.output, errors.New("du exited non-zero")
	}
	return r.Runner.Capture(name, args...)
}
