# Security Policy

## Reporting Security Vulnerabilities

Please do NOT open public GitHub issues for security vulnerabilities.

Instead, please report security vulnerabilities by emailing the maintainers privately with:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if you have one)

## Security Considerations

NetPulse is designed with security and privacy in mind:

✅ **HTTPS Only**
- All connections to speed test providers are encrypted with HTTPS
- No unencrypted data transmission

✅ **No Personal Data Collection**
- NetPulse does NOT collect personal information
- Does NOT track users or build profiles
- Does NOT send results to remote servers without explicit user consent

✅ **Local Results**
- Results are only saved locally when explicitly requested
- Users have full control over result data

✅ **No Elevated Privileges**
- NetPulse does NOT require elevated privileges (sudo/admin)
- Runs with standard user permissions

✅ **Minimal Dependencies**
- Limited external dependencies reduces attack surface
- Custom implementations of provider integrations
- Regular dependency updates

✅ **Open Source**
- Code is publicly available for review
- Community can audit for vulnerabilities
- Transparent development process

## Supported Versions

| Version | Status | Support |
|---------|--------|---------|
| 1.x | Active | Current development branch |

## Security Updates

Critical security updates will be released as soon as possible after discovery. Users are encouraged to:
- Keep NetPulse updated
- Subscribe to release notifications
- Report vulnerabilities responsibly

## Dependencies

NetPulse uses minimal external dependencies:
- `golang.org/x/net` - Standard library network extensions (official Go package)
- `github.com/urfave/cli/v2` - CLI framework (well-maintained, popular)

All dependencies are regularly reviewed and updated.

## Best Practices for Users

1. **Keep Updated**: Run `go install` regularly to get the latest version
2. **Review Results**: Check output before saving to files
3. **Use HTTPS**: NetPulse only connects via HTTPS
4. **Secure Results**: Protect saved result files if they contain sensitive info
5. **Report Issues**: Use responsible disclosure

## Vulnerability Disclosure Timeline

1. **Report**: Send security vulnerability report
2. **Acknowledgment**: We acknowledge receipt within 48 hours
3. **Assessment**: We assess impact and develop fix
4. **Patch**: Security patch is developed and tested
5. **Release**: Update is released and announced
6. **Publication**: Vulnerability is disclosed publicly

## Questions about Security?

For non-urgent security questions, open a GitHub Discussion in the "Security" category.

For urgent matters, contact maintainers directly.
