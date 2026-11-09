package main

import "testing"

func TestSelectGokartVersion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		version       string
		moduleVersion string
		want          string
	}{
		{name: "non-dev kept", version: "v1.2.3", moduleVersion: "v9.9.9", want: "v1.2.3"},
		{name: "dev with module version", version: "dev", moduleVersion: "v1.2.3", want: "v1.2.3"},
		{name: "dev without module version", version: "dev", moduleVersion: "", want: "dev"},
		{name: "dev devel marked", version: "dev", moduleVersion: "(devel)", want: "dev"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := selectGokartVersion(tc.version, tc.moduleVersion); got != tc.want {
				t.Errorf("selectGokartVersion(%q, %q) = %q, want %q", tc.version, tc.moduleVersion, got, tc.want)
			}
		})
	}
}
