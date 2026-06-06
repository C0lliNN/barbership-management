package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Barbearia do João", "barbearia-do-joao"},
		{"São Paulo", "sao-paulo"},
		{"  Hello World  ", "hello-world"},
		{"A & B + C", "a-b-c"},
		{"", ""},
		{"Café & Barbearia", "cafe-barbearia"},
		{"---test---", "test"},
		{"123 Main St.", "123-main-st"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, slugify(tc.input), "slugify(%q)", tc.input)
	}
}
