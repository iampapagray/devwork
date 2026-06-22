# Configuration reference

devwork reads two files and the environment.

## Files

### Global — `~/.config/devwork/config.toml` (secrets, `0600`)

Honors `$XDG_CONFIG_HOME`. Created from a template on first run.

```toml
# default_profile = "work"   # (phase 2)

[providers.jira]
base_url = "https://acme.atlassian.net"
email    = "me@acme.com"
token    = ""   # or DEVWORK_JIRA_TOKEN

[providers.github]
token = ""      # or DEVWORK_GITHUB_TOKEN / GITHUB_TOKEN / gh CLI

[providers.gitlab]
base_url = "https://gitlab.com"
token    = ""   # or DEVWORK_GITLAB_TOKEN
```

### Per-repo — `.devwork.toml` (committed, **no secrets**)

```toml
provider    = "jira"          # default provider for bare ids in this repo
template    = "{key}_{slug}"  # version gate OFF by default
base_branch = ""              # "" = current branch; a name; or "{default}"

[version]
source   = "package.json"     # only consulted if template has {version}
mismatch = ""                 # "" = warn+confirm; "strict" = hard abort

[slug]
max_words = 6
max_chars = 50
stopwords = ["the", "a", "an", "of", "to", "for"]
```

`.devwork.toml` is discovered by walking up from the current directory.

## Precedence (low → high)

built-in defaults → global `config.toml` → per-repo `.devwork.toml` → environment
variables → command-line flags.

## Template tokens

| Token        | Value                                            |
|--------------|--------------------------------------------------|
| `{key}`      | issue key (`PROJ-123`, or `123` for GH/GL)       |
| `{slug}`     | slugified issue title                            |
| `{version}`  | branch-prefix version (activates the version gate) |
| `{provider}` | provider name (`jira`/`github`/`gitlab`)         |

Unknown tokens are a configuration error.

## Version gate (when the template uses `{version}`)

The repo version comes from `[version].source`:

- `package.json` — the `.version` field (default)
- `VERSION` — the file's contents
- `git-tag` — `git describe --tags --abbrev=0` (leading `v` stripped)
- `auto` — try the above in order

It is normalized to `MAJOR.MINOR` with an **integer** minor (so `7.10 > 7.9`).
The issue's version (Jira fix version / GitHub/GitLab milestone) is the branch
prefix. Outcomes:

| Situation                                   | Result                                  |
|---------------------------------------------|-----------------------------------------|
| issue or repo version unresolvable          | hard error                              |
| equal                                       | proceed                                 |
| issue **ahead** of repo                     | confirm (post-release); issue version wins |
| issue **behind**/incompatible              | warn + confirm (or abort if `mismatch="strict"`) |

`--yes` skips the confirmation prompt but never bypasses a `strict` abort.

## Environment variables

| Variable                | Effect                                            |
|-------------------------|---------------------------------------------------|
| `DEVWORK_JIRA_TOKEN`    | overrides `[providers.jira].token`                |
| `DEVWORK_GITHUB_TOKEN`  | overrides `[providers.github].token`              |
| `DEVWORK_GITLAB_TOKEN`  | overrides `[providers.gitlab].token`              |
| `GITHUB_TOKEN`          | GitHub fallback (after `DEVWORK_GITHUB_TOKEN`)    |
| `XDG_CONFIG_HOME`       | base dir for the global config                    |
| `NO_COLOR`              | disable ANSI colors                               |

GitHub token order: `DEVWORK_GITHUB_TOKEN` → `GITHUB_TOKEN` → `gh auth token`
→ config file.
