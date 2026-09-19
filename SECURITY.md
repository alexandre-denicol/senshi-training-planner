# Security Policy

## Reporting a vulnerability

Please do not disclose security vulnerabilities through public GitHub issues.

Use GitHub's private vulnerability reporting feature when available, or contact the repository owner privately through the contact methods listed on the GitHub profile.

When reporting an issue, include:

- affected component or endpoint;
- reproduction steps;
- potential impact;
- relevant logs or screenshots with secrets removed.

Do not include passwords, session tokens, database credentials, or personal data in a report.

## Security principles

The project follows these baseline practices:

- secrets are supplied through environment variables and are not committed;
- passwords are stored using Argon2id hashes;
- authentication uses opaque server-side sessions;
- production authentication requires HTTPS;
- session cookies are HttpOnly;
- authorization checks are enforced by the backend;
- database access is restricted to the backend service;
- database accounts should use least privilege.

## Supported version

Security fixes are applied to the current default branch.
