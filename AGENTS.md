# AGENTS.md

> **For AI Agents:** This file contains critical context for understanding and working with the oauth-server codebase. Read this fully before making changes. Pay special attention to "Never Do" and "Common Agent Mistakes" sections.

OpenShift OAuth Server that handles user authentication and token issuance for OpenShift clusters.

## What This Component Does

The OAuth Server is the authentication gateway for OpenShift clusters, implementing OAuth 2.0 flows to authenticate users via configurable identity providers and issue access tokens. It manages sessions with encrypted cookies and stores tokens as Kubernetes CRDs via oauth-apiserver.

**Deployment**: `oauth-openshift` in `openshift-authentication` namespace, managed by cluster-authentication-operator

## How to Use This File

**Before making changes:**
1. Read "What This Component Does" to understand scope
2. Check "Never Do" and "Common Agent Mistakes" to avoid critical errors
3. Review "Architecture Patterns" for code structure
4. Check "Commands" for verification steps

**When stuck:**
1. Check "Common Workflows" for your specific task
2. Review "Key File Locations" to find relevant code
3. Check "Ask First" to see if you need human input
4. Review "Architecture & Design Documentation" for design context

## Related Documentation

- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - System design, flows, security, deployment details
- **[CONTRIBUTING.md](./CONTRIBUTING.md)** - Code conventions and PR process
- **[README.md](./README.md)** - Repository overview

## Architecture & Design Documentation

- **ARCHITECTURE.md**: Complete architecture documentation covering system context, authentication flows, key design decisions, security architecture, and deployment
- **Enhancement Proposals**: See [OpenShift enhancements repo](https://github.com/openshift/enhancements) for architectural decisions affecting authentication
- **OAuth 2.0 RFC**: https://datatracker.ietf.org/doc/html/rfc6749
- **OSIN Library**: https://github.com/openshift/osin (OpenShift's fork of OAuth 2.0 implementation)

**Note**: This is a server binary handling HTTP requests, not an operator. It does not use Kubernetes controller patterns.

## Commands

For complete build, test, and development commands, see [CONTRIBUTING.md - Development Setup](./CONTRIBUTING.md#development-setup).

**Quick Reference**:
```bash
# Build and test
make build                              # Build binary
make test-unit                          # Run all tests with race detection
make verify                             # Run verification checks

# Run locally (requires kubeconfig)
./oauth-server osinserver --config=/path/to/config.yaml --v=4
```

## Tech Stack

Check `go.mod` for current dependency versions. Key dependencies:
- **Go** - 1.24.0+
- **Kubernetes** - client-go, apiserver (OpenShift fork)
- **OpenShift** - library-go, api, osin (OAuth 2.0 implementation)
- **Logging** - klog/v2
- **CLI** - Cobra
- **Sessions** - gorilla/sessions
- **Auth** - go-ldap/ldap, go-jose/go-jose

## Always Do

- **Use structured logging with klog.V()** - Use appropriate log levels (V(2) for important info, V(4) for debug)
- **Validate IdP configurations** - All identity provider configs must be validated before use
- **Handle authentication errors securely** - Never leak sensitive information in error messages
- **Use table-driven tests** for unit tests - Follow existing test patterns
- **Return meaningful errors** - Wrap errors with context before returning
- **Propagate `context.Context`** - When a function accepts or has access to a `context.Context`, pass it through to downstream calls that accept one. Never discard a context or substitute `context.Background()`/`context.TODO()` when a context is already available
- **Reference files with line numbers** - Format: `pkg/oauthserver/auth.go:123`
- **Test across platforms** - Unit tests must pass on Linux and MacOS
- **Follow OAuth 2.0 spec** - When implementing OAuth flows, adhere to RFC 6749

## Ask First

- **Adding new identity provider types** - Major architectural change, requires design review
- **Modifying OAuth protocol flows** - Security-critical, must maintain OAuth 2.0 compliance
- **Changing token formats or encryption** - Impacts all clusters, requires careful migration
- **Modifying session cookie behavior** - Security implications, requires review
- **Changing API integration patterns** - Affects interaction with oauth-apiserver and Kubernetes API
- **Adding new dependencies** - Especially security-sensitive libraries (crypto, auth)
- **Modifying audit logging** - Compliance requirements, must maintain audit trail integrity

## Never Do

- **Never commit secrets** or credentials to the repo (including test credentials)
- **Never log sensitive data** - Passwords, tokens, session cookies, client secrets
- **Never modify `vendor/`** directly - Use `go get`, `go mod tidy`, and `go mod vendor`
- **Never skip authentication** - Even in "development mode" or test code paths
- **Never use generic "helpful assistant" code** - Follow OpenShift server patterns
- **Never store tokens in plaintext** - Tokens in etcd are encrypted at rest by the operator
- **Never bypass TLS validation** - Even in test code (use proper test certificates)
- **Never implement custom crypto** - Use standard libraries (golang.org/x/crypto, go-jose)
- **Never break OAuth 2.0 compliance** - Changes must conform to RFC 6749
- **Never modify OSIN library usage** without understanding OAuth 2.0 implications

## Common Agent Mistakes to Avoid

These are specific mistakes AI agents frequently make in this codebase:

- **Don't suggest adding controllers** - This is a server binary, not an operator. It handles HTTP requests, not Kubernetes resources via controllers.
- **Don't propose Kubernetes informers/listers for OAuth logic** - The server uses direct API calls for OAuth resources (tokens, clients), not watch-based controllers.
- **Don't recommend changing OAuth flow logic without OAuth 2.0 knowledge** - This implements RFC 6749. Changes must maintain compliance.
- **Don't suggest logging passwords or tokens** - Even in debug mode. This is a security violation.
- **Don't propose adding new IdP types without discussing storage/config** - Each IdP type requires config schema, authenticator implementation, and operator integration.
- **Don't modify OSIN usage without understanding the abstraction** - OSIN provides OAuth 2.0 storage abstraction. Direct storage access breaks this.
- **Don't suggest session storage in databases** - Sessions use encrypted cookies (gorilla/sessions). Session cookies are intentionally session-scoped (no explicit MaxAge set) and cleared when the browser closes.
- **Don't propose breaking changes to token format** - Existing tokens in production clusters must remain valid.
- **Don't recommend removing security headers** - HTTPOnly, Secure, SameSite cookies are security requirements.
- **Don't assume this is an apiserver for CRDs** - It's an OAuth server that happens to use apiserver libraries for HTTP handling.

## Architecture Patterns

This is **not** a Kubernetes operator. It's a server application that handles HTTP requests for OAuth flows.

### Server Pattern

The OAuth server is built on Kubernetes apiserver libraries but serves OAuth endpoints, not Kubernetes APIs:

```go
import (
    "k8s.io/apiserver/pkg/server"
    "github.com/openshift/osin"
)

// Server setup (pkg/oauthserver/)
func NewOAuthServer(config *Config) (*server.GenericAPIServer, error) {
    // Create generic API server
    genericServer, err := server.NewAPIServer(config)

    // Install OAuth endpoints (not Kubernetes API endpoints)
    InstallOAuthRoutes(genericServer.Handler.NonGoRestfulMux)

    return genericServer, nil
}
```

### OSIN (OAuth 2.0) Pattern

OAuth 2.0 protocol implementation uses the OSIN library (`pkg/osinserver/`):

```go
import "github.com/openshift/osin"

// OSIN server configuration
osinConfig := &osin.ServerConfig{
    AuthorizationExpiration:   300,    // 5 minutes
    AccessExpiration:          86400,  // 24 hours
    AllowedAuthorizeTypes:     osin.AllowedAuthorizeType{osin.CODE, osin.TOKEN},
    AllowedAccessTypes:        osin.AllowedAccessType{osin.AUTHORIZATION_CODE, osin.PASSWORD},
}

// Storage backend (adapts Kubernetes CRDs to OSIN interface)
storage := NewOAuthStorageAdapter(oauthClient)

// Create OSIN server
osinServer := osin.NewServer(osinConfig, storage)
```

### Authenticator Pattern

Identity providers implement the authenticator interface (`pkg/authenticator/`):

```go
type Password interface {
    AuthenticatePassword(ctx context.Context, username, password string) (*Response, bool, error)
}

type Request interface {
    AuthenticateRequest(req *http.Request) (*Response, bool, error)
}

// Authenticator response
type Response struct {
    User  *user.Info
    Success bool
    Err     error
}
```

### Session Pattern

Session management uses gorilla/sessions (`pkg/server/session/`):

```go
import "github.com/gorilla/sessions"

// Create session store with encryption key
store := sessions.NewCookieStore(sessionKey)
store.Options = &sessions.Options{
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteLaxMode,
    MaxAge:   300, // 5 minutes
}

// Use in handler
session, _ := store.Get(r, "ssn")
session.Values["user"] = userInfo
session.Save(r, w)
```

## Key File Locations

```
cmd/oauth-server/                    # Main entry point
pkg/cmd/oauth-server/                # CLI command implementation
pkg/oauthserver/                     # Main OAuth server setup
pkg/osinserver/                      # OSIN (OAuth 2.0) server implementation
pkg/authenticator/                   # Identity provider authenticators
  ├── password/                      # Password-based authenticators (HTPasswd, LDAP, etc.)
  ├── request/                       # Request-based authenticators (OIDC, OAuth, Request Header)
  ├── challenger/                    # Challenge handlers for WWW-Authenticate
  └── identitymapper/                # Maps external identities to OpenShift Users
pkg/oauth/                           # OAuth-specific logic
  ├── handlers/                      # OAuth flow handlers (authorize, token, grant)
  ├── registry/                      # OAuth resource registry (tokens, clients)
  └── external/                      # External OAuth provider handlers (GitHub, GitLab, Google)
pkg/server/                          # HTTP server components
  ├── login/                         # Login page handlers
  ├── grant/                         # Grant approval page handlers
  ├── session/                       # Session management
  ├── csrf/                          # CSRF protection
  └── crypto/                        # Cryptographic utilities
pkg/userregistry/                    # User/Identity resource management
pkg/audit/                           # Audit logging
pkg/api/                             # API client wrappers
test/files/                          # Test fixtures and data
```

## Common Workflows

### Add a New Identity Provider Type

1. Define authenticator in `pkg/authenticator/password/` (for password-based) or `pkg/authenticator/request/` (for request-based)
2. Implement the `Password` or `Request` interface
3. Add configuration parsing in `pkg/config/`
4. Add test coverage in `*_test.go`
5. Coordinate with cluster-authentication-operator team to add CRD schema support

Example:
```go
// pkg/authenticator/password/myidp/myidp.go
type MyIdPAuthenticator struct {
    // config fields
}

func (a *MyIdPAuthenticator) AuthenticatePassword(ctx context.Context, username, password string) (*authenticator.Response, bool, error) {
    // Validate against external IdP
    user, err := a.validateCredentials(username, password)
    if err != nil {
        return nil, false, err
    }

    return &authenticator.Response{
        User: &user.DefaultInfo{
            Name: username,
            UID:  user.ID,
        },
    }, true, nil
}
```

### Modify OAuth Flow Logic

**WARNING**: OAuth flows are security-critical. Changes must maintain OAuth 2.0 compliance.

1. Identify the flow in `pkg/osinserver/` or `pkg/oauth/handlers/`
2. Review OAuth 2.0 RFC 6749 for compliance requirements
3. Make changes preserving existing token formats and behaviors
4. Add comprehensive unit tests covering security edge cases
5. Test manually with real OAuth clients (oc login, web console)

### Add Audit Logging for New Event

1. Define audit event in `pkg/audit/`
2. Add logging call in the appropriate handler
3. Follow Kubernetes audit log format
4. Include relevant security context (user, IP, client)

Example:
```go
import "github.com/openshift/oauth-server/pkg/audit"

// In handler
audit.LogAuthenticationAttempt(req.Context(), username, idpName, success, reason)
```

### Debug Authentication Failures

```bash
# Increase log verbosity
./oauth-server osinserver --config=... --v=6

# Check audit logs
tail -f /var/log/oauth-server/audit.log | jq 'select(.verb == "authenticate")'

# Enable specific logger
# (In code, use klog.V(4).Infof() for debug messages)
```

## Security Notes

- **Authentication is the security boundary** - This component is the gatekeeper to the cluster
- **Token security is critical** - Access tokens grant full API access
- **Credentials must never be logged** - Even in debug mode, passwords/tokens are forbidden in logs
- **TLS everywhere** - All HTTP endpoints require HTTPS
- **Session cookies are security-critical** - Must be HTTPOnly, Secure, SameSite
- **Token encryption at rest** - Managed by cluster-authentication-operator, not this component
- **Audit all authentication attempts** - Required for security compliance (FIPS, FedRAMP)
- **Validate external IdP responses** - Never trust external data without validation
- **CSRF protection required** - All state-changing operations must have CSRF tokens
- **Follow OAuth 2.0 security best practices** - RFC 6749 Section 10 (Security Considerations)

## Testing Notes

### Unit Testing

- **Location**: Co-located `*_test.go` files in each package
- **Framework**: Go standard testing + `github.com/stretchr/testify`
- **Pattern**: Table-driven tests are preferred

Example:
```go
func TestMyAuthenticator(t *testing.T) {
    tests := []struct {
        name        string
        username    string
        password    string
        wantSuccess bool
        wantErr     bool
    }{
        {"valid credentials", "user1", "pass1", true, false},
        {"invalid password", "user1", "wrong", false, false},
        {"missing user", "unknown", "pass", false, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            auth := NewMyAuthenticator()
            resp, success, err := auth.AuthenticatePassword(context.TODO(), tt.username, tt.password)
            if success != tt.wantSuccess {
                t.Errorf("got success=%v, want %v", success, tt.wantSuccess)
            }
            if (err != nil) != tt.wantErr {
                t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
            }
        })
    }
}
```

### Integration & E2E Testing

- **Integration**: Full OAuth flows in `test/` directory (requires running cluster)
- **E2E**: Tests live in cluster-authentication-operator repo

## Configuration

The OAuth server reads configuration from a YAML file (mounted as ConfigMap). See [ARCHITECTURE.md - Configuration](./ARCHITECTURE.md#configuration) for details.

## Known Limitations

- OAuth 2.0 only (not OAuth 2.1, no PKCE)
- Based on unmaintained upstream OSIN library (OpenShift maintains fork)
- Session persistence lost on pod restart
- etcd-based token storage limits at very large scale

For deployment, performance, and external OIDC migration details, see [ARCHITECTURE.md](./ARCHITECTURE.md).

---

## Document Maintenance

**Last Updated**: 2026-06-23
**Review Frequency**: Quarterly or when major changes occur
**Owner**: OpenShift Control Plane - Authentication Team

**Update Triggers**:
- New identity provider type added
- OAuth 2.0 flow changes
- Security-critical changes
- Major dependency updates (Kubernetes, Go, OSIN)
- Deployment model changes
