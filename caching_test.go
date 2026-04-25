package githubauth

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProvider counts calls and returns a configured (token, expiresAt, err).
type fakeProvider struct {
	mu      sync.Mutex
	calls   int32
	tokens  []string  // returned in order; last value reused if exhausted
	expiry  time.Time // returned for every call
	failOn  int       // call number (1-indexed) on which to return the error
	failErr error
}

func (f *fakeProvider) GenerateToken(_ context.Context) (string, time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	atomic.AddInt32(&f.calls, 1)
	n := int(atomic.LoadInt32(&f.calls))

	if f.failOn != 0 && n == f.failOn {
		return "", time.Time{}, f.failErr
	}

	idx := n - 1
	if idx >= len(f.tokens) {
		idx = len(f.tokens) - 1
	}
	return f.tokens[idx], f.expiry, nil
}

func TestCachingProvider_FreshCacheReused(t *testing.T) {
	inner := &fakeProvider{
		tokens: []string{"first", "second"},
		expiry: time.Now().Add(45 * time.Minute),
	}
	cp := NewCachingProvider(inner)

	// First call: cache miss; mints "first".
	token, _, err := cp.GenerateToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "first", token)

	// Second call: still inside validity → cache hit; returns "first" again.
	token, _, err = cp.GenerateToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "first", token)

	assert.Equal(t, int32(1), atomic.LoadInt32(&inner.calls), "inner provider should be called once")
}

func TestCachingProvider_NearExpiryRefreshes(t *testing.T) {
	inner := &fakeProvider{
		tokens: []string{"first", "second"},
		expiry: time.Now().Add(2 * time.Minute), // < 5m default skew
	}
	cp := NewCachingProvider(inner)

	token, _, err := cp.GenerateToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "first", token)

	// First call cached "first" with expiry 2m out — that's already
	// within skew, so the next call should refresh and return "second".
	token, _, err = cp.GenerateToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "second", token)
	assert.Equal(t, int32(2), atomic.LoadInt32(&inner.calls))
}

func TestCachingProvider_CustomSkew(t *testing.T) {
	inner := &fakeProvider{
		tokens: []string{"first", "second"},
		expiry: time.Now().Add(10 * time.Minute),
	}
	// With a 30-minute skew, a 10-minute-out token is already considered
	// stale and triggers a refresh on every call.
	cp := NewCachingProvider(inner, WithRefreshSkew(30*time.Minute))

	_, _, _ = cp.GenerateToken(context.Background())
	_, _, _ = cp.GenerateToken(context.Background())
	assert.Equal(t, int32(2), atomic.LoadInt32(&inner.calls))
}
