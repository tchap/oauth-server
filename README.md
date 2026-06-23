# oauth-server

The OAuth Server is the authentication gateway for OpenShift clusters. It handles user authentication via configurable identity providers and issues OAuth 2.0 access tokens for cluster API access.

## Overview

The OAuth Server runs as the `oauth-openshift` deployment in the `openshift-authentication` namespace and is managed by the [cluster-authentication-operator](https://github.com/openshift/cluster-authentication-operator).

**Documentation**: See [ARCHITECTURE.md](./ARCHITECTURE.md) for comprehensive architecture details including features, components, and design decisions

## Quick Start

**View OAuth server in a cluster:**

```bash
oc get deployment oauth-openshift -n openshift-authentication
oc get pods -n openshift-authentication -l app=oauth-openshift
oc logs -n openshift-authentication -l app=oauth-openshift -c oauth-openshift
oc get oauth cluster -o yaml
```

**Build locally:**

```bash
make build
```

For development setup, build commands, testing, and project structure, see [CONTRIBUTING.md - Development Setup](./CONTRIBUTING.md#development-setup).

## Deployment

The OAuth server is deployed as `oauth-openshift` in the `openshift-authentication` namespace and is managed by the [cluster-authentication-operator](https://github.com/openshift/cluster-authentication-operator). **Do not deploy manually in production clusters.**

For complete deployment details including replicas, volumes, dependencies, and configuration, see [ARCHITECTURE.md - Deployment Architecture](./ARCHITECTURE.md#deployment-architecture).

## Security

The OAuth server is a security-critical component:
- Follow OAuth 2.0 security best practices (RFC 6749 Section 10)
- Never log passwords, tokens, or session cookies
- Never bypass TLS validation or skip authentication checks
- For security issues, follow the [OpenShift Security Response Process](https://github.com/openshift/security-response-process)

For detailed security architecture, see [ARCHITECTURE.md - Security Architecture](./ARCHITECTURE.md#security-architecture).

## External OIDC Migration

**Note**: External OIDC support is available; when enabled, this OAuth server is disabled and the cluster uses an external OIDC provider. See [ARCHITECTURE.md - External OIDC](./ARCHITECTURE.md#external-oidc-mode) for details.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for code conventions, testing guidelines, and pull request process.

## Resources

- **OpenShift Authentication Documentation**: https://docs.openshift.com/container-platform/latest/authentication/
- **OAuth 2.0 RFC 6749**: https://datatracker.ietf.org/doc/html/rfc6749
- **OpenID Connect Core 1.0**: https://openid.net/specs/openid-connect-core-1_0.html

## Support

For issues and questions:
- **Red Hat Employees**: Post in [#forum-ocp-apiserver](https://redhat.enterprise.slack.com/archives/CB48XQ4KZ) Slack channel
- **Bug Reports**: File a Jira ticket in the CNTRLPLANE project
- **Feature Requests**: File an RFE (Request for Enhancement) in Jira; approved RFEs may require an [OpenShift Enhancement Proposal](https://github.com/openshift/enhancements)
- **Security Issues**: Follow the [OpenShift Security Response Process](https://github.com/openshift/security-response-process)

## License

This project is licensed under the Apache License 2.0. See the LICENSE file for details.
