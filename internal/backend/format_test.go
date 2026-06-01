package backend

import "testing"

func TestIsGGUFModel(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"", false},
		{"model.gguf", true},
		{"model.GGUF", true},
		{"/path/to/Model.GgUf", true},
		{"model.safetensors", false},
		{"model.bin", false},
		{"some/path/snapshots/abc/", false},
	}
	for _, tc := range cases {
		if got := IsGGUFModel(tc.path); got != tc.want {
			t.Errorf("IsGGUFModel(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestIsSafeTensorsModel(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"", false},
		{"model.safetensors", true},
		{"model.SafeTensors", true},
		{"/path/snapshots/rev1/", true},
		{"/path/snapshots/rev1/file.safetensors", true},
		{"/path/no_snap_here/file.bin", false},
		{"model.gguf", false},
	}
	for _, tc := range cases {
		if got := IsSafeTensorsModel(tc.path); got != tc.want {
			t.Errorf("IsSafeTensorsModel(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
