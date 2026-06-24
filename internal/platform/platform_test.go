package platform

import "testing"

func TestResolveAlias(t *testing.T) {
	aliases := map[string]string{
		"arch":    "archlinux",
		"manjaro": "archlinux",
	}
	if got := ResolveAlias("manjaro", aliases); got != "archlinux" {
		t.Fatalf("ResolveAlias(manjaro) = %q; want archlinux", got)
	}
	if got := ResolveAlias("windows", aliases); got != "windows" {
		t.Fatalf("ResolveAlias(windows) = %q; want windows", got)
	}
}
