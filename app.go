package githubauth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// jwtExpiry is the lifetime of the JWT used to authenticate as the App.
	jwtExpiry = 10 * time.Minute

	// tokenRequestTimeout is the HTTP timeout for the token exchange.
	tokenRequestTimeout = 10 * time.Second

	// defaultAPIBaseURL targets github.com.
	defaultAPIBaseURL = "https://api.github.com"
)

// AppProvider mints short-lived installation access tokens for a GitHub
// App. Each call to GenerateToken signs a fresh JWT and exchanges it for
// an installation token via the GitHub REST API.
type AppProvider struct {
	appID          int64
	installationID int64
	privateKey     *rsa.PrivateKey
	apiBaseURL     string
	httpClient     *http.Client
}

// Compile-time check.
var _ TokenGenerator = (*AppProvider)(nil)

// WithAPIBaseURL overrides the GitHub API base URL. Trailing slashes are
// trimmed. An empty string is a no-op.
func WithAPIBaseURL(u string) Option {
	return func(p *AppProvider) {
		if u == "" {
			return
		}
		p.apiBaseURL = strings.TrimRight(u, "/")
	}
}

// NewAppProvider constructs an AppProvider by reading the PEM private
// key from disk.
func NewAppProvider(appID, installationID int64, privateKeyPath string, opts ...Option) (*AppProvider, error) {
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	p := &AppProvider{
		appID:          appID,
		installationID: installationID,
		privateKey:     key,
		apiBaseURL:     defaultAPIBaseURL,
		httpClient:     &http.Client{Timeout: tokenRequestTimeout},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

// NewAppProviderWithKey constructs an AppProvider from an already-parsed
// RSA key and a custom API base URL. Intended for testing.
func NewAppProviderWithKey(appID, installationID int64, key *rsa.PrivateKey, apiBaseURL string) (*AppProvider, error) {
	if key == nil {
		return nil, errors.New("private key is nil")
	}
	base := apiBaseURL
	if base == "" {
		base = defaultAPIBaseURL
	}
	return &AppProvider{
		appID:          appID,
		installationID: installationID,
		privateKey:     key,
		apiBaseURL:     strings.TrimRight(base, "/"),
		httpClient:     &http.Client{Timeout: tokenRequestTimeout},
	}, nil
}

// createJWT builds a signed JWT for authenticating as the GitHub App.
func (p *AppProvider) createJWT() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    strconv.FormatInt(p.appID, 10),
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)), // skew tolerance
		ExpiresAt: jwt.NewNumericDate(now.Add(jwtExpiry)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(p.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	return signed, nil
}
