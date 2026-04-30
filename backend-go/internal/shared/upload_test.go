package shared

import "testing"

func TestValidateFileName(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{name: "report.PDF", ok: true},
		{name: "archive.zip", ok: false},
		{name: "notes.md", ok: true},
		{name: "no-extension", ok: false},
	}

	for _, tc := range cases {
		if got := ValidateFileName(tc.name); got != tc.ok {
			t.Fatalf("ValidateFileName(%q) = %v, want %v", tc.name, got, tc.ok)
		}
	}
}

func TestValidateDataAssetFileName(t *testing.T) {
	if !ValidateDataAssetFileName("dataset.bin") {
		t.Fatal("expected non-empty data asset file name to be accepted")
	}
	if ValidateDataAssetFileName(" \t ") {
		t.Fatal("expected blank data asset file name to be rejected")
	}
}
