# Jira

## Configure

```toml
[providers.jira]
base_url = "https://acme.atlassian.net"
email    = "me@acme.com"
token    = ""   # or DEVWORK_JIRA_TOKEN
```

## Token

Create an **API token** at
<https://id.atlassian.com/manage-profile/security/api-tokens>. Jira Cloud uses
HTTP Basic auth with your account email + the API token (not your password).

## Input forms

- Issue key: `devwork QA-2840`
- Browse URL: `devwork https://acme.atlassian.net/browse/QA-2840`

The key is upper-cased; the first `ABC-123`-shaped token in the input is used.

## Version

The normalized version is the first `fixVersions[].name` that parses to
`MAJOR.MINOR`. With a `{version}` template, set a fix version on the issue or
the gate hard-errors.
