# Sandbox [![CI](https://github.com/servusdei2018/sandbox/actions/workflows/test.yml/badge.svg)](https://github.com/servusdei2018/sandbox/actions)

Run coding agents—like Claude Code, Gemini CLI, Codex, OpenCode, or go, python and node, safely isolated in Docker containers. 

`sandbox` automatically bind-mounts your current directory into a fresh container, protects your secrets, and cleans up after itself.

## Why Sandbox?

When you give AI coding agents open access to your terminal, mistakes can happen. Code can get comprimised. Secrets leaked.

`sandbox` wraps those agents in a Docker container so they can read and write to your workspace without having the keys to your entire operating system. It's fast, automatic, and tries to be secure by default.

- 🐳 **Invisible Docker** — You run a command, we handle the container.
- 📁 **Auto-mounting** — Your current directory is instantly available at `/work`.
- 🔐 **Secret scrubbing** — Because your AWS keys and GitHub tokens shouldn't be handed to an AI model by default.
- 🤖 **Smart detection** — We know which base image you need. Run `sandbox run python` and you get Python. Run `sandbox run claude` and you get Claude.
- ⚡ **Lightning fast** — If the image is pulled, you're running in less than 2 seconds.
- ♻️ **Ephemeral** — Containers disappear when they exit. No clutter.

## Getting Started

### Prerequisites
- Go 1.21 or newer (to build)
- Docker Desktop, OrbStack, Rancher Desktop, or Podman

### Installation

```bash
git clone https://github.com/servusdei2018/sandbox
cd sandbox
make build-prod
sudo mv bin/sandbox /usr/local/bin/
```

## How to use it

It's as simple as prepending `sandbox run` to whatever command you want to execute safely.

```bash
# General commands
sandbox run echo "Hello from inside the box"
sandbox run sh -c "ls /work"

# Agents (we'll automatically pull the right image)
sandbox run claude
sandbox run gemini --help

# Languages and tools
sandbox run python -c "print('hello')"
sandbox run node -e "console.log('hello')"
sandbox run bun run index.ts
```

### Advanced Usage

```bash
# Need a specific version? Override the image:
sandbox run --image python:3.11-slim python app.py

# Got a long-running task? Dial up the timeout:
sandbox run --timeout 15m python train.py

# Need to poke around after a crash? Keep the container:
sandbox run --keep sh
```

You can view your current configuration with `sandbox config show` or generate a fresh config file with `sandbox config init`.

## Supported Agents & Runtimes

Out of the box, `sandbox` automatically detects and routes the following tools to their appropriate base images:

**Coding Agents:**
- Claude Code (`claude`)
- Gemini CLI (`gemini`)
- Codex (`codex`)
- Kilocode (`kilo`, `kilocode`)
- OpenCode (`opencode`)

**Runtimes & Package Managers:**
- Python (`python`, `python3`, `pip`, `pip3`)
- Node.js (`node`, `npm`, `npx`)
- Bun (`bun`, `bunx`)
- Go (`go`)

*(If a command doesn't match these, it falls back to a generic Alpine Linux image.)*

## Configuration

On its first run, `sandbox` generates a configuration file at `~/.sandbox/config.yaml`. It looks like this:

```yaml
# Sandbox CLI Configuration
# See https://github.com/servusdei2018/sandbox for documentation.

images:
    bun: oven/bun:alpine
    claude: ghcr.io/servusdei2018/sandbox-claude:latest
    codex: ghcr.io/servusdei2018/sandbox-codex:latest
    default: alpine:latest
    gemini: ghcr.io/servusdei2018/sandbox-gemini:latest
    go: golang:1.26-alpine
    kilocode: ghcr.io/servusdei2018/sandbox-kilocode:latest
    node: node:24-alpine
    opencode: ghcr.io/servusdei2018/sandbox-opencode:latest
    python: python:3.13-alpine
env_whitelist:
    - LANG
    - LC_ALL
    - LC_CTYPE
    - SHELL
    - TERM
    - COLORTERM
    - XTERM_VERSION
    - TZ
env_blocklist:
    - AWS_ACCESS_KEY_ID
    - AWS_SECRET_ACCESS_KEY
    - AWS_SESSION_TOKEN
    - AWS_*
    - GCP_*
    - GOOGLE_APPLICATION_CREDENTIALS
    - GITHUB_TOKEN
    - GIT_PASSWORD
    - ANTHROPIC_API_KEY
    - OPENAI_API_KEY
    - COHERE_API_KEY
container:
    timeout: 30m
    network_mode: bridge
    remove: true
logging:
    level: info
    format: console
paths:
    workspace: /work
    config_dir: ~/.sandbox
    cache_dir: ~/.sandbox/cache
```

## Developing

Want to contribute? 

```bash
make help              # Show all available targets
make build             # Build debug binary to ./bin/sandbox
make build-prod        # Build production binary
make test              # Run unit tests
make test-integration  # Run Docker integration tests
make lint              # Run golangci-lint
make fmt               # Format the code
```

## License

MIT License. See [LICENSE](LICENSE) for details.
