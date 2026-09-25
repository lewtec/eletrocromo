package common

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProductName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id, name, want string
	}{
		{"br.tec.lew.counter", "Counter", "Counter"},
		{"br.tec.lew.counter", "My App", "My-App"},
		{"br.tec.lew.counter", "", "counter"},
		{"br.tec.lew.x", "!!!", "x"},
	}
	for _, tt := range tests {
		t.Run(tt.want+"/"+tt.name, func(t *testing.T) {
			t.Parallel()
			got := ProductName(tt.id, tt.name)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultOutApp(t *testing.T) {
	t.Parallel()
	got := DefaultOutApp("Counter", "/tmp/proj")
	want := filepath.Join("/tmp/proj", "dist", "Counter.app")
	assert.Equal(t, want, got)
	got = DefaultOutApp("Counter.app", "/tmp/proj")
	assert.Equal(t, want, got)
}

func TestXMLEscape(t *testing.T) {
	t.Parallel()
	got := XMLEscape(`A & B <C> "d"`)
	want := `A &amp; B &lt;C&gt; &quot;d&quot;`
	assert.Equal(t, want, got)
}
