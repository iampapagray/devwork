# Changelog

All notable changes are documented here. This project follows
[Semantic Versioning](https://semver.org/) and
[Conventional Commits](https://www.conventionalcommits.org/).

## [Unreleased]

### Added
- Initial Go rewrite of the original bash `devwork` script.
- Multi-tracker support: **Jira**, **GitHub**, and **GitLab** via a pluggable
  provider interface + registry.
- TOML configuration: global `~/.config/devwork/config.toml` (secrets, `0600`)
  and committed per-repo `.devwork.toml` (behavior). First run writes a
  template and exits with guidance.
- Configurable branch templates (`{key}`, `{slug}`, `{version}`, `{provider}`).
  The version gate is active only when the template uses `{version}`; the
  default `{key}_{slug}` never aborts on version grounds.
- Credential resolution via `DEVWORK_*_TOKEN` env vars and the config file;
  GitHub also honors `GITHUB_TOKEN` and the `gh` CLI.
- Flags: `--task/-t` (and positional), `--provider`, `--from`, `--yes/-y`,
  `--dry-run`, `--push`, plus shell `completion`.
- `jira-strict` preset reproducing the original `v{version}/{key}_{slug}`
  behavior with strict version matching.
- Distribution via GoReleaser (macOS/Linux/Windows), a Homebrew tap, and a
  `curl | sh` installer with checksum verification.

### Changed (breaking vs. the bash script)
- The legacy `-task` single-dash flag is dropped; use `--task/-t` or the
  positional `devwork PROJ-123`.
