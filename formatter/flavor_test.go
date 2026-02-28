package formatter

import "testing"

func TestParseFlavor(t *testing.T) {
	tests := []struct {
		input   string
		want    MarkdownFlavor
		wantErr bool
	}{
		{"gfm", FlavorGFM, false},
		{"commonmark", FlavorCommonMark, false},
		{"", "", true},
		{"foo", "", true},
		{"GFM", "", true},
		{"COMMONMARK", "", true},
		{"Gfm", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFlavor(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseFlavor(%q): expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseFlavor(%q): unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFlavor(%q): got %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
