package tui

import "testing"

func TestFormatByteSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{size: 12, want: "12 B"},
		{size: 1024, want: "1.0 KB"},
		{size: 1536, want: "1.5 KB"},
		{size: 1024 * 1024, want: "1.0 MB"},
	}

	for _, tt := range tests {
		if got := formatByteSize(tt.size); got != tt.want {
			t.Fatalf("formatByteSize(%d) = %q, want %q", tt.size, got, tt.want)
		}
	}
}

func TestFileSizeStatus(t *testing.T) {
	tests := []struct {
		name  string
		sizes []int64
		want  string
	}{
		{name: "none", want: "none"},
		{name: "single", sizes: []int64{1536}, want: "1.5 KB"},
		{name: "multiple", sizes: []int64{1024, 2048}, want: "3.0 KB total, latest 2.0 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{fileSizes: tt.sizes}
			if got := m.fileSizeStatus(); got != tt.want {
				t.Fatalf("fileSizeStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
