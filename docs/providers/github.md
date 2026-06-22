# GitHub

## Configure

```toml
[providers.github]
token = ""   # or DEVWORK_GITHUB_TOKEN / GITHUB_TOKEN / gh CLI
```

Token resolution order: `DEVWORK_GITHUB_TOKEN` → `GITHUB_TOKEN` →
`gh auth token` → config file. If you use the [`gh` CLI](https://cli.github.com)
and are logged in, no token configuration is needed.

## Token / scopes

A fine-grained or classic PAT with **read access to issues** on the target
repositories. Private repos require `repo` (classic) or the equivalent
fine-grained "Issues: read" permission.

## Input forms

- Bare number (uses the `origin` remote for owner/repo): `devwork 42`, `devwork '#42'`
- Explicit: `devwork acme/widget#42`
- URL: `devwork https://github.com/acme/widget/issues/42` (also `/pull/`)

The branch key is the bare number (no `#`).

## Version

The normalized version is the issue's `milestone.title` when it parses to
`MAJOR.MINOR`; otherwise the issue has no version.
