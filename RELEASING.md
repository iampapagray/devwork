# Releasing

Releases are cut by GoReleaser from a `v*` git tag via
[`.github/workflows/release.yml`](.github/workflows/release.yml). Binaries are
published to the GitHub Release and the Homebrew formula is pushed to the tap.

## One-time setup

1. **Create the tap repo.** A **public** repo named exactly `homebrew-tap`
   under your account: `github.com/iampapagray/homebrew-tap`.
   - Homebrew maps the short form `iampapagray/tap` →
     `github.com/iampapagray/homebrew-tap`, so the name must be `homebrew-tap`.
   - It can start empty (initialize with a README so it's clonable). GoReleaser
     creates and updates `Formula/devwork.rb` on `main` automatically.

2. **Create a token for the tap.** A Personal Access Token with
   **`contents: write`** on `iampapagray/homebrew-tap`:
   - Fine-grained PAT: scope it to the `homebrew-tap` repo, "Contents:
     Read and write".
   - Classic PAT: `repo` scope.

3. **Add the secret.** In `iampapagray/devwork` →
   *Settings → Secrets and variables → Actions*, add the token as
   **`HOMEBREW_TAP_GITHUB_TOKEN`**.
   - The default `GITHUB_TOKEN` is **not** enough — it can only write to the
     repo running the workflow, not the separate tap repo.

## Cutting a release

```sh
# Land your changes on main, then:
git tag v1.0.0
git push origin v1.0.0
```

The tag push triggers `release.yml`, which runs `goreleaser release --clean`:

- builds macOS (arm64/amd64), Linux (arm64/amd64), and Windows (amd64) binaries,
- bundles shell completions, `presets/`, `LICENSE`, and `README.md` into the
  archives,
- writes `checksums.txt`,
- publishes the GitHub Release with a changelog from Conventional Commits,
- pushes the updated `Formula/devwork.rb` to the tap (which installs `devwork`
  and symlinks `devw`).

Use [Semantic Versioning](https://semver.org/) tags and
[Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`,
…) so the generated changelog is meaningful.

## Verifying

After the workflow succeeds:

```sh
brew install iampapagray/tap/devwork   # or: brew upgrade devwork
devwork --version

# curl|sh fallback:
curl -fsSL https://raw.githubusercontent.com/iampapagray/devwork/main/install.sh | sh
```

## Dry run / validation

- `goreleaser check` validates the config (also run in CI's `goreleaser-check`).
- `goreleaser release --snapshot --clean` builds locally into `dist/` without
  tagging, publishing, or touching the tap — handy for sanity-checking archives.
