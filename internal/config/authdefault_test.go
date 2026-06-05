package config

import "testing"

// TestOAuthLoginID pins the provider→login-id mapping derived from the provider
// auth methods. Providers without an OAuth method must return "".
func TestOAuthLoginID(t *testing.T) {
	want := map[string]string{
		"anthropic":  "anthropic",
		"openai":     "openai-codex",
		"openrouter": "openrouter",
		"minimax":    "",
		"mimo":       "",
		"unknown":    "",
	}
	for p, id := range want {
		if got := OAuthLoginID(p); got != id {
			t.Errorf("OAuthLoginID(%q) = %q, want %q", p, got, id)
		}
	}
}

// TestReorderAuthMethods checks that the preferred kind is promoted ahead of the
// rest while preserving the declared order otherwise.
func TestReorderAuthMethods(t *testing.T) {
	methods := AuthMethodsFor("anthropic") // [apikey, claude-oauth-token(bearer), oauth-token]

	// No preference: order unchanged.
	if got := reorderAuthMethods(methods, ""); !sameOrder(got, methods) {
		t.Errorf("no-pref reorder changed order")
	}

	// Prefer OAuth: the two OAuth methods come first, API key last.
	oauthFirst := reorderAuthMethods(methods, AuthDefaultOAuth)
	if isOAuthMethod(oauthFirst[len(oauthFirst)-1]) {
		t.Errorf("prefer-oauth: expected API-key method last, got OAuth method")
	}
	if !isOAuthMethod(oauthFirst[0]) {
		t.Errorf("prefer-oauth: expected an OAuth method first")
	}

	// Prefer API key: an API-key (non-OAuth) method leads.
	apiFirst := reorderAuthMethods(methods, AuthDefaultAPIKey)
	if isOAuthMethod(apiFirst[0]) {
		t.Errorf("prefer-api_key: expected an API-key method first")
	}

	// Single-method provider is unaffected regardless of preference.
	mm := AuthMethodsFor("minimax")
	if got := reorderAuthMethods(mm, AuthDefaultOAuth); !sameOrder(got, mm) {
		t.Errorf("single-method provider should be unaffected by preference")
	}
}

func sameOrder(a, b []AuthMethod) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Kind != b[i].Kind || a[i].EnvVar != b[i].EnvVar || a[i].LoginID != b[i].LoginID || a[i].HeaderStyle != b[i].HeaderStyle {
			return false
		}
	}
	return true
}

// TestEffectiveDefault covers the default-method derivation when no explicit
// preference is stored, and that an explicit preference wins.
func TestEffectiveDefault(t *testing.T) {
	cases := []struct {
		pref   string
		apiKey bool
		oauth  bool
		want   string
	}{
		{"", false, false, AuthDefaultAPIKey},
		{"", true, false, AuthDefaultAPIKey},
		{"", false, true, AuthDefaultOAuth},
		{"", true, true, AuthDefaultAPIKey},
		{AuthDefaultOAuth, true, false, AuthDefaultOAuth},
		{AuthDefaultAPIKey, false, true, AuthDefaultAPIKey},
	}
	for _, c := range cases {
		if got := effectiveDefault(c.pref, c.apiKey, c.oauth); got != c.want {
			t.Errorf("effectiveDefault(%q,%v,%v) = %q, want %q", c.pref, c.apiKey, c.oauth, got, c.want)
		}
	}
}
