package tui

import (
	"math"
	"testing"
)

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want string
	}{
		{name: "unknown", size: 0, want: "size unknown"},
		{name: "bytes", size: 512, want: "512 B"},
		{name: "kilobytes", size: 1024, want: "1.0 KB"},
		{name: "megabytes", size: 5 * 1024 * 1024, want: "5.0 MB"},
		{name: "maximum int64", size: math.MaxInt64, want: "8.0 EB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatFileSize(tt.size); got != tt.want {
				t.Fatalf("formatFileSize(%d) = %q, want %q", tt.size, got, tt.want)
			}
		})
	}
}
