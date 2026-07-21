package version

import "testing"

func TestResolve(t *testing.T) {
	tests := []struct {
		name     string
		explicit string
		module   string
		want     string
	}{
		{name: "ldflags release", explicit: "v0.2.0", module: "(devel)", want: "v0.2.0"},
		{name: "go install tag", explicit: "dev", module: "v0.2.0", want: "v0.2.0"},
		{name: "local devel", explicit: "dev", module: "(devel)", want: "dev"},
		{name: "empty explicit module tag", explicit: "", module: "v0.1.2", want: "v0.1.2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolve(tc.explicit, tc.module); got != tc.want {
				t.Fatalf("resolve(%q, %q) = %q, want %q", tc.explicit, tc.module, got, tc.want)
			}
		})
	}
}
