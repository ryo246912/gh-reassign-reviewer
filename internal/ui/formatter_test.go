package ui

import (
	"testing"
)

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{
			name:     "pad short string",
			input:    "hello",
			width:    10,
			expected: "hello     ",
		},
		{
			name:     "no padding needed",
			input:    "hello",
			width:    5,
			expected: "hello",
		},
		{
			name:     "string longer than width",
			input:    "hello world",
			width:    5,
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			width:    5,
			expected: "     ",
		},
		{
			name:     "zero width",
			input:    "hello",
			width:    0,
			expected: "hello",
		},
		{
			name:     "unicode characters",
			input:    "こんにちは",
			width:    15,
			expected: "こんにちは     ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PadRight(tt.input, tt.width)
			if got != tt.expected {
				t.Errorf("PadRight(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.expected)
			}
		})
	}
}
