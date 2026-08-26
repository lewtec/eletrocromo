package main

import "testing"

func TestInboxNameFromURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"eletrocromo-inbox://inbox?name=photo.jpg", "photo.jpg"},
		{"eletrocromo-inbox://inbox?file=a.png", "a.png"},
		{"eletrocromo-inbox://inbox?path=../x.txt", "x.txt"},
		{"eletrocromo-inbox://inbox/foo.md", "foo.md"},
		{"eletrocromo-inbox:///inbox/foo.md", "foo.md"},
		{"eletrocromo-inbox:inbox/bar.txt", "bar.txt"},
		{"eletrocromo-inbox://from-webview", ""},
		{"eletrocromo-inbox://x/y?q=1", ""},
		{"://no-scheme", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := inboxNameFromURL(tt.in); got != tt.want {
				t.Fatalf("inboxNameFromURL(%q)=%q want %q", tt.in, got, tt.want)
			}
		})
	}
}
