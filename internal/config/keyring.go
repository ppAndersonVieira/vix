package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/zalando/go-keyring"

	"github.com/get-vix/vix/internal/auth"
)

const (
	keyringService = "vix"
)

// Auth-default kinds: which credential method a provider prefers when more than
// one is available. Stored as a small non-secret marker in the OS keychain
// (see ProviderAuthDefault) so the preference lives alongside the credentials
// it governs and is honored by credential resolution everywhere.
const (
	AuthDefaultAPIKey = "api_key"
	AuthDefaultOAuth  = "oauth"
)

// KeySource describes where the API key was found.
type KeySource string

const (
	KeySourceEnv        KeySource = "env"
	KeySourceOAuthToken KeySource = "oauth-token"
	KeySourceKeychain   KeySource = "keychain"
	KeySourceEnvFile    KeySource = "dotenv"
	KeySourceNone       KeySource = "none"
)

// Credential bundles an API key or OAuth token with everything an adapter
// needs to authenticate: its source (for display), the wire header style, an
// optional endpoint override, and any extra headers implied by the auth method.
type Credential struct {
	Value        string
	Source       KeySource
	HeaderStyle  HeaderStyle       // "" (APIKeyHeader) means SDK-native API key header
	BaseURL      string            // "" means use the adapter's default endpoint
	ExtraHeaders map[string]string // applied verbatim on every request
}

// RequestOptions returns the appropriate Anthropic SDK options for this
// credential, driven by HeaderStyle and ExtraHeaders. BearerHeader sends an
// Authorization: Bearer token; otherwise the value is sent as the API key
// (x-api-key on Anthropic).
func (c Credential) RequestOptions() []option.RequestOption {
	var opts []option.RequestOption
	if c.HeaderStyle == BearerHeader {
		opts = append(opts, option.WithHeader("Authorization", "Bearer "+c.Value))
	} else {
		opts = append(opts, option.WithAPIKey(c.Value))
	}
	for k, v := range c.ExtraHeaders {
		opts = append(opts, option.WithHeader(k, v))
	}
	return opts
}

// ResolveEnvVar checks the environment and .env files for a variable.
// Returns the value and true if found, or empty string and false.
func ResolveEnvVar(name string) (string, bool) {
	if v := os.Getenv(name); v != "" {
		return v, true
	}
	if v := loadKeyFromEnvFile(loadExeEnvFilePath(), name); v != "" {
		return v, true
	}
	if v := loadKeyFromEnvFile(".env", name); v != "" {
		return v, true
	}
	return "", false
}

// ProviderKey holds a provider name and a display prefix of its stored key.
type ProviderKey struct {
	Provider string
	Prefix   string // first 10 chars of the stored key, for display; empty if not stored
}

// providerKeyringUser returns the keyring "user" field for storing a provider's
// primary API key, e.g. "anthropic" → "anthropic-api-key". Used by the
// store/delete/list helpers; credential resolution uses AuthMethod.Keyring.
func providerKeyringUser(provider string) string {
	return provider + "-api-key"
}

// resolveKey searches env var, OS keychain, and .env files for the given variable name
// and optional keyring user. Returns the value and source, or empty if not found.
func resolveKey(envVar, keyringUser string) (string, KeySource) {
	// 1. Environment variable
	if envVar != "" {
		if key := os.Getenv(envVar); key != "" {
			return key, KeySourceEnv
		}
	}

	// 2. OS Keychain
	if keyringUser != "" {
		if key, err := keyring.Get(keyringService, keyringUser); err == nil && key != "" {
			return key, KeySourceKeychain
		}
	}

	// 3. .env next to executable
	if envVar != "" {
		if key := loadKeyFromEnvFile(loadExeEnvFilePath(), envVar); key != "" {
			return key, KeySourceEnvFile
		}

		// 4. .env in CWD
		if key := loadKeyFromEnvFile(".env", envVar); key != "" {
			return key, KeySourceEnvFile
		}
	}

	return "", KeySourceNone
}

// ResolveProviderCredential resolves a Credential for the given provider by
// walking its AuthMethods (see providers.go) in priority order and returning
// the first that yields a value. API-key methods are resolved env → keychain →
// .env (env-first); OAuth methods are resolved from the keychain-backed auth
// subsystem (without refreshing). Returns a KeySourceNone credential when
// nothing is found.
func ResolveProviderCredential(provider string) Credential {
	return resolveProviderCredential(context.Background(), provider, false)
}

// ResolveProviderCredentialFresh behaves like ResolveProviderCredential but
// refreshes an expired stored OAuth access token first (bounded by ctx).
func ResolveProviderCredentialFresh(ctx context.Context, provider string) Credential {
	return resolveProviderCredential(ctx, provider, true)
}

func resolveProviderCredential(ctx context.Context, provider string, refresh bool) Credential {
	for _, m := range orderedAuthMethods(provider) {
		switch m.Kind {
		case APIKeyAuth:
			value, src := resolveKey(m.EnvVar, m.Keyring)
			if value == "" {
				continue
			}
			if m.HeaderStyle == BearerHeader {
				src = KeySourceOAuthToken
			}
			return buildCredential(value, src, m)
		case OAuthMintKey, OAuthToken:
			value := resolveOAuthValue(ctx, m.LoginID, refresh)
			if value == "" {
				continue
			}
			return buildCredential(value, KeySourceOAuthToken, m)
		}
	}
	return Credential{Source: KeySourceNone}
}

// isOAuthMethod reports whether a method obtains its credential via an OAuth
// login. This includes the interactive OAuth flows (OAuthMintKey/OAuthToken)
// and bearer-style API-key methods, which carry an OAuth token shipped through
// env/keychain (e.g. CLAUDE_CODE_OAUTH_TOKEN).
func isOAuthMethod(m AuthMethod) bool {
	if m.Kind == OAuthMintKey || m.Kind == OAuthToken {
		return true
	}
	return m.Kind == APIKeyAuth && m.HeaderStyle == BearerHeader
}

// orderedAuthMethods returns a provider's auth methods reordered so the user's
// preferred kind (api_key or oauth) is tried first. When no preference is
// stored, the provider's declared method order (providers.json) is preserved.
func orderedAuthMethods(provider string) []AuthMethod {
	return reorderAuthMethods(AuthMethodsFor(provider), ProviderAuthDefault(provider))
}

// reorderAuthMethods is the pure core of orderedAuthMethods: it promotes the
// methods matching the preferred kind ahead of the rest while keeping the
// relative order within each group stable, so resolution still falls through to
// the other method when the preferred one yields no value.
func reorderAuthMethods(methods []AuthMethod, pref string) []AuthMethod {
	if pref == "" || len(methods) < 2 {
		return methods
	}
	preferOAuth := pref == AuthDefaultOAuth
	preferred := make([]AuthMethod, 0, len(methods))
	rest := make([]AuthMethod, 0, len(methods))
	for _, m := range methods {
		if isOAuthMethod(m) == preferOAuth {
			preferred = append(preferred, m)
		} else {
			rest = append(rest, m)
		}
	}
	return append(preferred, rest...)
}

// buildCredential assembles a Credential from a resolved value and its auth
// method (header style, endpoint override, and any derived extra headers).
func buildCredential(value string, src KeySource, m AuthMethod) Credential {
	cred := Credential{Value: value, Source: src, HeaderStyle: m.HeaderStyle, BaseURL: m.BaseURL}
	if m.Extra != nil {
		cred.ExtraHeaders = m.Extra(value)
	}
	return cred
}

// resolveOAuthValue returns the stored access token / minted key for an OAuth
// login id, optionally refreshing an expired token. Returns "" when no login is
// stored or resolution fails (callers then fall through to the next method).
func resolveOAuthValue(ctx context.Context, loginID string, refresh bool) string {
	if loginID == "" {
		return ""
	}
	st := auth.DefaultStorage()
	if refresh {
		tok, err := st.AccessTokenRefreshing(ctx, loginID)
		if err != nil {
			return ""
		}
		return tok
	}
	tok, _, ok := st.AccessToken(loginID)
	if !ok {
		return ""
	}
	return tok
}

// ResolveProviderKey returns the resolved credential value and source for a
// provider. Thin wrapper over ResolveProviderCredential for callers that only
// need the key string.
func ResolveProviderKey(provider string) (key string, source KeySource) {
	cred := ResolveProviderCredential(provider)
	return cred.Value, cred.Source
}

// StoreProviderKey writes the API key for the given provider to the OS keychain.
func StoreProviderKey(provider, key string) error {
	return keyring.Set(keyringService, providerKeyringUser(provider), key)
}

// DeleteProviderKey removes the API key for the given provider from the OS keychain.
// It also clears any stored default-method marker so credential resolution falls
// back to whatever credential remains.
func DeleteProviderKey(provider string) error {
	err := keyring.Delete(keyringService, providerKeyringUser(provider))
	_ = ClearProviderAuthDefault(provider)
	return err
}

// providerAuthDefaultUser returns the keyring "user" field for a provider's
// default-method marker, e.g. "anthropic" → "anthropic-auth-default".
func providerAuthDefaultUser(provider string) string {
	return provider + "-auth-default"
}

// ProviderAuthDefault returns the stored default-method preference for a
// provider — AuthDefaultAPIKey, AuthDefaultOAuth, or "" when unset.
func ProviderAuthDefault(provider string) string {
	v, err := keyring.Get(keyringService, providerAuthDefaultUser(provider))
	if err != nil {
		return ""
	}
	return v
}

// SetProviderAuthDefault stores the default-method preference for a provider.
func SetProviderAuthDefault(provider, kind string) error {
	return keyring.Set(keyringService, providerAuthDefaultUser(provider), kind)
}

// ClearProviderAuthDefault removes a provider's default-method marker. A missing
// marker is not an error.
func ClearProviderAuthDefault(provider string) error {
	err := keyring.Delete(keyringService, providerAuthDefaultUser(provider))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// ProviderAuthStatus summarizes a provider's stored credentials and which method
// is the effective default. It is the single read the UI needs to render the
// authentication panel without touching the keychain or auth subsystem directly.
type ProviderAuthStatus struct {
	APIKeyStored   bool
	APIKeyPrefix   string // first 10 chars of the stored API key, for display
	OAuthStored    bool
	OAuthSupported bool   // provider has an OAuth login method
	Default        string // AuthDefaultAPIKey | AuthDefaultOAuth (effective)
}

// GetProviderAuthStatus reports the stored-credential state and effective
// default method for a provider. When no preference is stored, the default is
// derived: API key if one exists, else OAuth if a token exists, else API key.
func GetProviderAuthStatus(provider string) ProviderAuthStatus {
	st := ProviderAuthStatus{}
	if k, err := keyring.Get(keyringService, providerKeyringUser(provider)); err == nil && k != "" {
		st.APIKeyStored = true
		if len(k) > 10 {
			st.APIKeyPrefix = k[:10]
		} else {
			st.APIKeyPrefix = k
		}
	}
	if loginID := OAuthLoginID(provider); loginID != "" {
		st.OAuthSupported = true
		st.OAuthStored = auth.DefaultStorage().HasLogin(loginID)
	}
	switch ProviderAuthDefault(provider) {
	case AuthDefaultOAuth:
		st.Default = AuthDefaultOAuth
	case AuthDefaultAPIKey:
		st.Default = AuthDefaultAPIKey
	default:
		st.Default = effectiveDefault("", st.APIKeyStored, st.OAuthStored)
	}
	return st
}

// effectiveDefault derives the default auth method when there is no explicit
// preference: API key if one is stored, else OAuth if a token is stored, else
// API key. An explicit pref ("api_key"/"oauth") is returned verbatim.
func effectiveDefault(pref string, apiKeyStored, oauthStored bool) string {
	switch pref {
	case AuthDefaultOAuth:
		return AuthDefaultOAuth
	case AuthDefaultAPIKey:
		return AuthDefaultAPIKey
	}
	if apiKeyStored {
		return AuthDefaultAPIKey
	}
	if oauthStored {
		return AuthDefaultOAuth
	}
	return AuthDefaultAPIKey
}

// ListStoredProviderKeys returns the stored key info for all known providers.
// The Prefix field holds the first 10 chars of the stored key (empty if not stored).
func ListStoredProviderKeys() []ProviderKey {
	providers := KnownProviders()
	result := make([]ProviderKey, 0, len(providers))
	for _, p := range providers {
		pk := ProviderKey{Provider: p}
		if k, err := keyring.Get(keyringService, providerKeyringUser(p)); err == nil && k != "" {
			if len(k) > 10 {
				pk.Prefix = k[:10]
			} else {
				pk.Prefix = k
			}
		}
		result = append(result, pk)
	}
	return result
}

// loadExeEnvFilePath returns the path to the .env file next to the executable.
func loadExeEnvFilePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "..", "..", ".env")
}

// loadKeyFromEnvFile reads a .env file and extracts the value of the given variable name.
func loadKeyFromEnvFile(path, varName string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	prefix := varName + "="
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimPrefix(strings.TrimSpace(line), "export ")
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.SplitN(line, "=", 2)[1])
		}
	}
	return ""
}
