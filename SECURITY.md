# Security Policy

## Reporting A Vulnerability

Do not open a public issue for suspected security vulnerabilities or include
credentials, OAuth tokens, email content, or other sensitive data in public
reports.

Report vulnerabilities privately through GitHub Security Advisories:

<https://github.com/ajramos/giztui/security/advisories/new>

Include the affected version, impact, reproduction steps, and any proposed
mitigation that can be shared safely. You should receive an acknowledgement
within seven days. Public disclosure and release timing will be coordinated
after the issue is understood and a fix is available.

## Supported Versions

Security fixes target the latest published release. Users should upgrade to the
latest release before reporting an issue that may already be fixed.

## Sensitive Local Data

GizTUI stores OAuth credentials, tokens, configuration, and local cache/database
files under `~/.config/giztui/`. Never attach those files to a public issue.
