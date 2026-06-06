package identity

import (
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// slugify converts a display name to a URL-safe slug.
// "Barbearia do João" → "barbearia-do-joao"
func slugify(s string) string {
	// NFD-decompose so combining diacritics become separate runes, then strip them.
	t := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	}), norm.NFC)
	normalized, _, _ := transform.String(t, s)

	var b strings.Builder
	prevHyphen := true // leading hyphens are skipped
	for _, r := range strings.ToLower(normalized) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen {
			b.WriteByte('-')
			prevHyphen = true
		}
	}

	result := b.String()
	// Trim trailing hyphen.
	return strings.TrimRight(result, "-")
}
