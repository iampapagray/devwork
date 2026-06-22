# Presets

## `jira-strict`

Reproduces the behavior of the original bash `devwork` script: a
version-prefixed branch where the Jira fix version is authoritative and a fix
version older than the repo's `package.json` is a hard abort.

Branch format: `v{version}/{ISSUE-KEY}_{slug}`
(e.g. `v7.3/QA-2840_alert-followups-and-close-modal-update`).

Apply it by copying [`presets/jira-strict.toml`](../presets/jira-strict.toml)
into your repo as `.devwork.toml`:

```toml
provider    = "jira"
template    = "v{version}/{key}_{slug}"

[version]
source   = "package.json"
mismatch = "strict"
```

### Behavior

- **Equal** fix version / repo version → proceed.
- **Ahead** (post-release transition) → confirm; the issue's fix version is the
  prefix.
- **Behind**/incompatible → hard abort (likely the wrong repo/branch).

### Difference from the bash script

The normalized model uses the **first** parseable fix version on the issue (see
the data-model mapping in the plan). If an issue carries several fix versions,
devwork keys off the first that parses to `MAJOR.MINOR` rather than searching
for the one matching `package.json`. Single-fix-version issues — the common
case — behave identically to the original script.
