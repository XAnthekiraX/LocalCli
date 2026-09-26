package context

import "testing"

// TestEstimarTokensEstable — la estimación es estable y monótona.
func TestEstimarTokensEstable(t *testing.T) {
	casos := []struct {
		texto  string
		quiero int
	}{
		{"", 0},
		{"a", 1},
		{"abcd", 1},
		{"abcde", 2},
	}
	for _, c := range casos {
		if got := EstimarTokens(c.texto); got != c.quiero {
			t.Errorf("EstimarTokens(%q) = %d, quiero %d", c.texto, got, c.quiero)
		}
	}
	if EstimarTokens("aaaaaaaaaa") <= EstimarTokens("aaaaa") {
		t.Error("más texto debe estimar más tokens")
	}
}
