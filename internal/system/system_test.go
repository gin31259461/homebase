package system

import "testing"

func TestParseInstalledPackages(t *testing.T) {
	got := ParseInstalledPackages("git\nbase-devel\n  go\n")
	for _, pkg := range []string{"git", "base-devel", "go"} {
		if !got[pkg] {
			t.Fatalf("missing package %q in %#v", pkg, got)
		}
	}
}

func TestContainsWord(t *testing.T) {
	if !ContainsWord("abner wheel docker", "docker") {
		t.Fatal("expected docker group")
	}
	if ContainsWord("abner wheel openrazer", "razer") {
		t.Fatal("expected exact word match")
	}
}
