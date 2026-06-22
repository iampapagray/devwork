# devwork

Turn a tracker issue into a well-named git branch — across **Jira, GitHub, and
GitLab** — with one command.

```console
$ devwork PROJ-123
  Base branch  main (current)
  Task         PROJ-123 — Fix login redirect loop
  New branch   PROJ-123_fix-login-redirect-loop
Create this branch from main? [Y/e/n] y
✓ created and switched to PROJ-123_fix-login-redirect-loop
```

## Install

### Homebrew (recommended)

```sh
brew install iampapagray/tap/devwork
```

`brew upgrade devwork` to update, `brew uninstall devwork` to remove. The
formula also installs a `devw` shorthand.

### curl | sh (Linux / CI fallback)

```sh
curl -fsSL https://raw.githubusercontent.com/iampapagray/devwork/main/install.sh | sh
```

Pin a version with `DEVWORK_VERSION=v1.0.0`, or set `PREFIX` to choose the
install dir (default `~/.local/bin`, falling back from `/usr/local/bin`). The
script verifies the release checksum and symlinks `devw`.

### Windows

Download `devwork_<ver>_windows_amd64.zip` from
[Releases](https://github.com/iampapagray/devwork/releases), extract
`devwork.exe` (and `devw.exe`), and add the folder to your `PATH`.

## Quickstart

1. Run `devwork` once. With no credentials it writes a template to
   `~/.config/devwork/config.toml` (mode `0600`) and exits with guidance.
2. Fill in a provider (or set a `DEVWORK_*_TOKEN` env var — see
   [docs/config.md](docs/config.md)).
3. From inside a git repo:

```sh
devwork PROJ-123                       # Jira issue key
devwork https://github.com/o/r/issues/42
devwork --provider gitlab 17 --push    # create and push -u origin
devwork QA-9 --dry-run                 # resolve + print, don't create
```

## How the branch name is built

The per-repo `.devwork.toml` (committed, no secrets) sets a template:

```toml
provider = "jira"            # default provider for bare ids in this repo
template = "{key}_{slug}"    # default — never aborts on version grounds
```

Tokens: `{key}`, `{slug}`, `{version}`, `{provider}`. The **version gate** runs
only when the template contains `{version}` — see
[docs/config.md](docs/config.md) and [docs/presets.md](docs/presets.md). The
original Jira behavior (`v{version}/{key}_{slug}`, strict version match) ships
as the [`jira-strict`](presets/jira-strict.toml) preset.

## Documentation

- [docs/config.md](docs/config.md) — full config reference, precedence, env vars
- [docs/presets.md](docs/presets.md) — the strict/Jira preset
- [docs/providers/jira.md](docs/providers/jira.md),
  [github.md](docs/providers/github.md),
  [gitlab.md](docs/providers/gitlab.md) — tokens, scopes, env vars
- [docs/adding-a-provider.md](docs/adding-a-provider.md) — implement a new tracker
- [RELEASING.md](RELEASING.md) — cut a release (tap setup, tagging, verification)

## License

MIT — see [LICENSE](LICENSE).
