package paths

import "testing"

func TestCleanJoinParentBase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/Show/OldNs/Materials/", "/Show/OldNs/Materials"},
		{"/Show//OldNs/./Mat", "/Show/OldNs/Mat"},
		{"", ""},
		{"/", "/"},
	}
	for _, tc := range cases {
		if got := Clean(tc.in); got != tc.want {
			t.Errorf("Clean(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
	if got := Join("/Show/OldNs", "Materials", "Red"); got != "/Show/OldNs/Materials/Red" {
		t.Fatalf("Join=%q", got)
	}
	if got := Base("/Show/OldNs/Materials/Red"); got != "Red" {
		t.Fatalf("Base=%q", got)
	}
	if parent, ok := Parent("/Show/OldNs/Red"); !ok || parent != "/Show/OldNs" {
		t.Fatalf("Parent=%q ok=%v", parent, ok)
	}
	if _, ok := Parent("/"); ok {
		t.Fatal("root must have no parent")
	}
}

func TestHasPrefixComponentBoundary(t *testing.T) {
	old := "/Show/OldNamespace"
	yes := []string{
		"/Show/OldNamespace",
		"/Show/OldNamespace/Materials/Red",
		"/Show/OldNamespace/A/B/C",
	}
	for _, p := range yes {
		if !HasPrefix(p, old) {
			t.Errorf("expected %q under %q", p, old)
		}
	}
	no := []string{
		"/Show/OldNamespaceArchive",
		"/Show/OldNamespace2/Red",
		"/Show/OldNamespaceExtra",
		"/Other/OldNamespace",
	}
	for _, p := range no {
		if HasPrefix(p, old) {
			t.Errorf("did not expect %q under %q (component boundary)", p, old)
		}
	}
}

func TestRewritePrefixDescendants(t *testing.T) {
	const oldRoot = "/Show/OldNamespace"
	const newRoot = "/Show/Shot042/Set"
	cases := []struct {
		in, want string
		hit      bool
	}{
		{oldRoot, newRoot, true},
		{oldRoot + "/Materials/RedPlastic", newRoot + "/Materials/RedPlastic", true},
		{oldRoot + "/Cameras/HeroCam", newRoot + "/Cameras/HeroCam", true},
		{"/Show/Outside/Keep", "/Show/Outside/Keep", false},
		{"/Show/OldNamespaceArchive/X", "/Show/OldNamespaceArchive/X", false},
	}
	for _, tc := range cases {
		got, hit := RewritePrefix(oldRoot, newRoot, tc.in)
		if got != tc.want || hit != tc.hit {
			t.Errorf("RewritePrefix(%q)=(%q,%v) want (%q,%v)", tc.in, got, hit, tc.want, tc.hit)
		}
	}
}

func TestRawStringReplaceCorruptsBoundaryLookalike(t *testing.T) {
	// Documents why literal string replacement is unsafe; the migrator must
	// use RewritePrefix instead.
	got := RawStringReplace("/Show/OldNamespace", "/Show/New", "/Show/OldNamespaceArchive/M")
	if want := "/Show/NewArchive/M"; got != want {
		t.Fatalf("raw replace=%q want %q", got, want)
	}
	if rewritten, hit := RewritePrefix("/Show/OldNamespace", "/Show/New", "/Show/OldNamespaceArchive/M"); hit || rewritten != "/Show/OldNamespaceArchive/M" {
		t.Fatalf("component rewrite must leave look-alike alone: %q hit=%v", rewritten, hit)
	}
}
