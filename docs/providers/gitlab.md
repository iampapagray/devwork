# GitLab

## Configure

```toml
[providers.gitlab]
base_url = "https://gitlab.com"   # or your self-hosted instance
token    = ""                     # or DEVWORK_GITLAB_TOKEN
```

## Token / scopes

A **personal (or project) access token** with the `read_api` scope. Create one
under *Preferences → Access Tokens* on your GitLab instance.

## Input forms

- Bare iid (uses the `origin` remote for the project path): `devwork 17`, `devwork '#17'`
- Explicit project path (subgroups allowed): `devwork group/sub/proj#17`
- URL: `devwork https://gitlab.com/group/proj/-/issues/17`

A URL pins the host, so self-hosted instances work without extra config. The
branch key is the bare iid.

## Version

The normalized version is the issue's `milestone.title` when it parses to
`MAJOR.MINOR`; otherwise the issue has no version.
