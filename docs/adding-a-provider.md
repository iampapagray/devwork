# Adding a provider

A provider is a Go package implementing `provider.Provider` and registering
itself in `init()`. No central switch needs editing.

## Checklist

1. **New package** under `internal/provider/<name>/`.

2. **Implement the interface** (`internal/provider/provider.go`):

   ```go
   type Provider interface {
       Name() string
       Match(input string) MatchResult                    // confidence input is ours
       Resolve(ctx, input string, repo RepoContext) (issue.IssueRef, error) // pure: parse + owner/repo
       Fetch(ctx, ref issue.IssueRef, c creds.Credentials) (issue.Issue, error) // network
   }
   ```

   - `Match` returns `Strong` for an unambiguous signal (your host in a URL, a
     distinctive key shape) and `Weak` for an ambiguous shape (a bare number).
   - `Resolve` does parsing and owner/repo inference only — no HTTP. This is the
     `--dry-run` stop point and where most unit tests live.
   - `Fetch` makes the API call and normalizes into `issue.Issue`. Populate
     `Version` from your milestone/fix-version when it parses to `MAJOR.MINOR`;
     leave it `nil` otherwise (issues with no version simply can't use a
     `{version}` template).

3. **Register in `init()`**:

   ```go
   func init() { provider.Register(New()) }
   ```

4. **Blank-import** the package in `cmd/devwork/main.go`:

   ```go
   _ "github.com/iampapagray/devwork/internal/provider/<name>"
   ```

5. **Tests**: table-driven `Resolve` tests, plus a `httptest.Server` with a
   recorded JSON fixture for `Fetch`. See the jira/github/gitlab adapters.

6. **Docs**: add `docs/providers/<name>.md` (token creation, scopes, env vars)
   and update the credential resolver in `internal/creds` if the provider needs
   non-default auth handling.

## Credentials

`creds.EnvFileResolver` maps a provider name to its `config.ProviderConfig` and
applies fallbacks. Add a `case "<name>"` there for required-field validation or
extra token sources (as GitHub does for `GITHUB_TOKEN` / the `gh` CLI).
