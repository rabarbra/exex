package binfile

import "testing"

func TestTristateString(t *testing.T) {
	tests := map[Tristate]string{
		TriUnknown:  "unknown",
		TriYes:      "yes",
		TriNo:       "no",
		Tristate(9): "unknown",
	}
	for in, want := range tests {
		if got := in.String(); got != want {
			t.Fatalf("%v.String = %q, want %q", uint8(in), got, want)
		}
	}
}

func TestInfoHelpers(t *testing.T) {
	if got := splitColon([]string{"/a:/b", " /c ", ""}); len(got) != 3 || got[0] != "/a" || got[1] != "/b" || got[2] != "/c" {
		t.Fatalf("splitColon = %#v", got)
	}
	for in, want := range map[int]int{0: 0, 1: 4, 4: 4, 5: 8} {
		if got := align4(in); got != want {
			t.Fatalf("align4(%d) = %d, want %d", in, got, want)
		}
	}
	if got := indexBytes([]byte("abc musl libc xyz"), "musl"); got != 4 {
		t.Fatalf("indexBytes = %d, want 4", got)
	}
	if got := indexBytes([]byte("abc"), "missing"); got != -1 {
		t.Fatalf("missing indexBytes = %d, want -1", got)
	}
}

func TestExtractLibcVersions(t *testing.T) {
	glibc := []byte("GNU C Library stable release version 2.35. Copyright")
	if got := extractGlibcVersion(glibc); got != "2.35" {
		t.Fatalf("extractGlibcVersion = %q, want 2.35", got)
	}
	musl := []byte("banner musl libc (x86_64) Version 1.2.4\n")
	if got := extractMuslVersion(musl); got != "1.2.4" {
		t.Fatalf("extractMuslVersion = %q, want 1.2.4", got)
	}
	if got := extractGlibcVersion([]byte("not glibc")); got != "" {
		t.Fatalf("missing glibc version = %q", got)
	}
	if got := extractMuslVersion([]byte("not musl")); got != "" {
		t.Fatalf("missing musl version = %q", got)
	}
}
