# gha

[![CI](https://github.com/haribote-lab/github-app-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/haribote-lab/github-app-cli/actions/workflows/ci.yml)
[![Release](https://github.com/haribote-lab/github-app-cli/actions/workflows/release.yml/badge.svg)](https://github.com/haribote-lab/github-app-cli/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Proxy `gh` CLI commands with GitHub App authentication. Transparently generates an installation token from your GitHub App credentials and injects it into any `gh` command.

## Why?

GitHub Apps provide fine-grained, scoped permissions — ideal for CI/CD, automation, and bot workflows. But using them with the `gh` CLI requires manually generating JWTs, exchanging them for installation tokens, and passing them around. `gha` does all of that for you.

## Install

### Homebrew (macOS / Linux)

```bash
brew install haribote-lab/tap/gha
```

### Ubuntu / Debian

Download the `.deb` package from [Releases](https://github.com/haribote-lab/github-app-cli/releases) and install with `dpkg`:

```bash
VERSION=0.1.0
curl -sLO "https://github.com/haribote-lab/github-app-cli/releases/download/v${VERSION}/gha_${VERSION}_linux_amd64.deb"
sudo dpkg -i "gha_${VERSION}_linux_amd64.deb"
```

### From GitHub Releases

Download the binary from [Releases](https://github.com/haribote-lab/github-app-cli/releases) and place it in your `PATH`:

```bash
VERSION=0.1.0
curl -sL "https://github.com/haribote-lab/github-app-cli/releases/download/v${VERSION}/gha_${VERSION}_linux_amd64.tar.gz" | tar xz
sudo mv gha /usr/local/bin/
```

### From Source

```bash
go install github.com/haribote-lab/github-app-cli@latest
```

## Setup

```bash
gha configure
```

You will be prompted for:

| Field | Description |
|---|---|
| **App ID** | Your GitHub App's ID (Settings → Developer settings → GitHub Apps) |
| **Installation ID** | Optional. Press Enter to auto-detect (works when the App has a single installation) |
| **Private Key Path** | Absolute path to the `.pem` private key file |

If Installation ID is omitted, `gha` automatically resolves it via the GitHub API at runtime. If the App is installed on multiple organizations, you must specify the Installation ID explicitly.

Configuration is saved to `~/.config/github-app-cli/config.yaml` (respects `XDG_CONFIG_HOME`).

### Per-directory rules

You can switch installations automatically based on the current working directory — handy when running coding agents (Claude Code, Codex, etc.) across multiple projects. Add a `directories` section to `~/.config/github-app-cli/config.yaml`:

```yaml
app_id: 12345
private_key_path: ~/.config/github-app-cli/app.pem

directories:
  - path: ~/work/orgA
    installation_id: 11111
  - path: ~/work/orgB
    org: orgB
  - path: ~/work/orgA/special-repo
    installation_id: 22222
```

When `gha` runs, it walks up from the current directory and picks the longest matching `path`. In the example above, running inside `~/work/orgA/special-repo/subdir` uses installation `22222`, while `~/work/orgA/other` uses `11111`.

Resolution precedence (highest to lowest):

1. `--installation-id` / `--org` flag
2. `GHA_INSTALLATION_ID` / `GHA_ORG` env var
3. `directories` rule matching cwd
4. `installation_id` in config.yaml
5. Auto-detect (single installation only)

## Usage

Use `gha` exactly like `gh` — all arguments are passed through:

```bash
gha pr list
gha issue create --title "Bug" --body "Details"
gha api repos/{owner}/{repo}
gha repo clone owner/repo
```

Under the hood, `gha`:

1. Reads your GitHub App credentials from the config
2. Generates a short-lived JWT (RS256, 10-minute expiry)
3. Exchanges the JWT for an installation access token via the GitHub API
4. Sets `GH_TOKEN` and execs `gh` with your arguments

## How It Works

```
gha pr list
  │
  ├─ Load config (~/.config/github-app-cli/config.yaml)
  ├─ Read private key (.pem)
  ├─ Generate JWT (iat: now-30s, exp: now+10m, iss: app_id)
  ├─ POST /app/installations/{id}/access_tokens → installation token
  └─ exec gh pr list  (with GH_TOKEN=<installation_token>)
```

## License

MIT
