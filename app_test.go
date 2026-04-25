package githubauth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// genTestKey produces a 2048-bit RSA key for use in tests.
func genTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

// writeKeyToTempFile encodes key as PEM and returns the path of a
// temp file containing it. The file is auto-removed at test end.
func writeKeyToTempFile(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "private-key.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	require.NoError(t, os.WriteFile(path, pemBytes, 0o600))
	return path
}

func TestNewAppProvider_OK(t *testing.T) {
	key := genTestKey(t)
	path := writeKeyToTempFile(t, key)

	p, err := NewAppProvider(123, 456, path)
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestNewAppProvider_BadKeyPath(t *testing.T) {
	_, err := NewAppProvider(123, 456, "/does/not/exist/key.pem")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read private key")
}

func TestNewAppProvider_BadKeyContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.pem")
	require.NoError(t, os.WriteFile(path, []byte("not a pem"), 0o600))

	_, err := NewAppProvider(123, 456, path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse private key")
}

func TestNewAppProviderWithKey_NilKey(t *testing.T) {
	_, err := NewAppProviderWithKey(123, 456, nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key is nil")
}

func TestAppProvider_CreateJWT(t *testing.T) {
	key := genTestKey(t)
	p, err := NewAppProviderWithKey(123, 456, key, "")
	require.NoError(t, err)

	jwt, err := p.createJWT()
	require.NoError(t, err)
	assert.NotEmpty(t, jwt)
	// JWT format: three base64 segments separated by dots.
	dots := 0
	for _, c := range jwt {
		if c == '.' {
			dots++
		}
	}
	assert.Equal(t, 2, dots, "JWT should have exactly two dots")
}
