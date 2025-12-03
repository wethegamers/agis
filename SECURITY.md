# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report vulnerabilities via one of these methods:

1. **GitHub Security Advisories** (Preferred)  
   Use [GitHub's private vulnerability reporting](https://github.com/wethegamers/agis/security/advisories/new) to submit a detailed report.

2. **Email**  
   Send details to: `security@wethegamers.org`

### What to Include

Please include the following in your report:

- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact assessment
- Any suggested fixes (optional but appreciated)

### What to Expect

- **Acknowledgment**: We will acknowledge receipt within 48 hours
- **Initial Assessment**: We will provide an initial assessment within 7 days
- **Resolution Timeline**: Critical vulnerabilities will be prioritized for immediate patching
- **Disclosure**: We follow coordinated disclosure practices. We ask that you:
  - Allow us reasonable time to address the issue before public disclosure
  - Avoid exploiting the vulnerability beyond what's necessary to demonstrate it

### Safe Harbor

We consider security research conducted in accordance with this policy to be:

- Authorized and welcome
- Exempt from legal action under computer fraud laws
- Exempt from DMCA claims related to circumvention

We will not pursue legal action against researchers who:

- Act in good faith
- Avoid privacy violations, destruction of data, or service interruption
- Report vulnerabilities promptly and allow reasonable time for remediation

## Security Best Practices

When deploying AGIS Bot:

1. **Secrets Management**: Always use HashiCorp Vault or Kubernetes Secrets for sensitive data
2. **Network Security**: Deploy behind Tailscale or similar zero-trust network
3. **RBAC**: Follow least-privilege principles for Kubernetes service accounts
4. **Updates**: Keep dependencies current and monitor for security advisories

## License Considerations

This project is licensed under BSL-1.1. While the source is available for review, please note that security fixes may be applied to the licensed work before any public disclosure.
