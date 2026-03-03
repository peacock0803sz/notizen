# notizen

Personal notes and diary manager for the command line.

## Prerequisites

- Go 1.25+ to build from source
- git for `commit` and `specs` (skippable with `--name`)
- rsync for `sync`

## Installation

### Pre-built binaries

Grab a tarball from [GitHub Releases](https://github.com/peacock0803sz/notizen/releases) (linux and darwin, amd64 and arm64), extract it, and drop the binary into your `$PATH`:

```sh
tar xzf notizen-darwin-arm64.tar.gz
mv notizen-darwin-arm64 /usr/local/bin/notizen
```

### Build from source

```sh
go install github.com/peacock0803sz/notizen/cmd/notizen@latest
```

## Quick Start

```sh
notizen diary      # create today's diary entry
notizen commit     # stage, commit, and push changes
notizen sync       # rsync source/ to a remote server
```

## Commands

### `notizen diary`

Creates today's diary entry at `~/.notizen/source/Diaries/YYYY/MM/DD.md`, along with year and month directories and their index files.

| Flag | Description |
|------|-------------|
| `--no-mkdir` | Do not create missing directories |

### `notizen commit`

Fetches from origin, pulls, stages everything under `source/`, commits with a timestamped message, and pushes. Does nothing when there are no changes.

### `notizen sync`

Rsyncs `source/` to a remote server.

| Flag | Description |
|------|-------------|
| `-s`, `--src` | Local source directory (default: `~/.notizen/source`) |
| `-d`, `--dest` | Remote destination path (falls back to config) |
| `-n`, `--dry-run` | Preview without making changes |
| `--no-delete` | Keep remote files not present in source |
| `--no-recursive` | Non-recursive sync |

### `notizen specs <repo-path>`

Symlinks a repository's `specs/` directory into `source/Agents/Specs/<repo-name>/`.

| Flag | Description |
|------|-------------|
| `-n`, `--name` | Override the auto-detected repo name |

## Configuration

Reads `~/.config/notizen/config.toml`, or `$XDG_CONFIG_HOME/notizen/config.toml` if set.

```toml
[remote]
host = "example.com"
user = "deploy"
port = 22
key  = "/home/you/.ssh/id_ed25519"
path = "/var/www/notes"
```

Environment variables take precedence over the config file:

| Variable | Field |
|----------|-------|
| `NOTIZEN_REMOTE_HOST` | `host` |
| `NOTIZEN_REMOTE_USER` | `user` |
| `NOTIZEN_REMOTE_PORT` | `port` |
| `NOTIZEN_REMOTE_KEY` | `key` |
| `NOTIZEN_REMOTE_PATH` | `path` |

## Template Customization

Put override templates under `~/.config/notizen/templates/`. They take precedence over the built-in defaults.

Built-in templates:

- `diary/entry.md.tmpl` -- diary entry body
- `diary/index.md.tmpl` -- year and month index
- `speckit/branch.md.tmpl` -- spec branch page
- `speckit/index.md.tmpl` -- spec index page

## Directory Structure

```
~/.notizen/
  source/
    Diaries/
      YYYY/
        index.md
        MM/
          index.md
          DD.md
    Agents/
      Specs/
        <repo-name>/  -> /path/to/repo/specs/

~/.config/notizen/
  config.toml
  templates/
    diary/
    speckit/
```

## License

[MIT](LICENSE)
