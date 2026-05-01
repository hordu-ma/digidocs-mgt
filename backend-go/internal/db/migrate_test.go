package db

import "testing"

func TestExtractVersion(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{name: "001_initial_schema.sql", want: 1},
		{name: "012.sql", want: 12},
		{name: "20260430_add_code_repositories.sql", want: 20260430},
		{name: "no_version.sql", want: -1},
		{name: "_missing.sql", want: -1},
	}

	for _, tc := range cases {
		if got := extractVersion(tc.name); got != tc.want {
			t.Fatalf("extractVersion(%q) = %d, want %d", tc.name, got, tc.want)
		}
	}
}
