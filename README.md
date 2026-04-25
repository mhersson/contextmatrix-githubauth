# contextmatrix-githubauth

A small Go module providing GitHub authentication primitives shared by
[contextmatrix](https://github.com/mhersson/contextmatrix) and
[contextmatrix-runner](https://github.com/mhersson/contextmatrix-runner).

## Overview

```go
type TokenGenerator interface {
    GenerateToken(ctx context.Context) (token string, expiresAt time.Time, err error)
}
```

Three implementations:

- **`AppProvider`** — mints short-lived (~1h) installation access tokens
  from a GitHub App. Use this in production where possible.
- **`PATProvider`** — returns a static fine-grained personal access
  token with a sentinel "never refresh" expiry.
- **`CachingProvider`** — wraps another `TokenGenerator` and amortizes
  minting across calls. Suitable for server-side use where every call
  uses the token immediately. Not recommended in places where the token
  is handed off to a long-lived consumer (see the contextmatrix-runner
  rationale).

## Usage

```go
import "github.com/mhersson/contextmatrix-githubauth"

// GitHub App
inner, err := githubauth.NewAppProvider(
    appID, installationID, "/etc/secrets/github-app-key.pem",
    githubauth.WithAPIBaseURL("https://api.github.com"),
)
if err != nil { /* ... */ }

provider := githubauth.NewCachingProvider(inner)

// Or PAT
inner, err := githubauth.NewPATProvider(os.Getenv("GH_TOKEN"))
provider := githubauth.NewCachingProvider(inner)

// Use it
token, _, err := provider.GenerateToken(ctx)
req.Header.Set("Authorization", "Bearer "+token)
```

## Setup guides

For instructions on creating a GitHub App or fine-grained PAT and the
permissions each consumer requires, see the
[contextmatrix GitHub auth setup guide](https://github.com/mhersson/contextmatrix/blob/main/docs/github-auth-setup.md).

## License

MIT
