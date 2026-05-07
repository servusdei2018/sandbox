<p align="center">
  <img src=".github/assets/sandbox.svg" alt="Sandbox Logo" width="400">
</p>

# Sandbox &nbsp; [![CI](https://github.com/servusdei2018/sandbox/actions/workflows/test.yml/badge.svg)](https://github.com/servusdei2018/sandbox/actions)

Sandbox lets you run coding agents like Claude Code, Gemini CLI, and Codex, as well as runtimes like Go, Python, and Node, all within the safety of isolated Docker containers.

Sandbox automatically maps your current directory into a fresh container, protects your secrets, and cleans up after itself.

## Why Sandbox?

AI coding agents are powerful, but giving them full access to your terminal is risky. A small mistake can compromise your code or leak sensitive secrets.

Sandbox keeps these agents inside a secure Docker container, allowing them to work in your workspace without exposing your entire system. It’s fast, automatic, and designed to be secure from the start.

- **Invisible Docker**: You run your commands, and we manage the container lifecycle for you.
- **Automatic Mounting**: Your current directory is mapped directly to /work inside the container.
- **Secret Management**: We help protect your AWS keys and GitHub tokens so they aren't shared with AI models by default.
- **Smart Detection**: We automatically pick the right environment for you. If you run "sandbox run python," you get a Python environment.
- **High Performance**: Once the image is downloaded, your environment starts in less than two seconds.
- **Ephemeral Environments**: All containers are cleaned up automatically as soon as they're no longer needed.

## Getting Started

### Prerequisites
- Go 1.21 or newer (to build)
- Docker Desktop, OrbStack, Rancher Desktop, or Podman

### Installation

#### From Source

```bash
git clone https://github.com/servusdei2018/sandbox
cd sandbox
make build-prod
sudo mv bin/sandbox /usr/local/bin/
```

#### Via Go

```bash
go install github.com/servusdei2018/sandbox/cmd/sandbox@latest
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

# Use a custom seccomp profile:
sandbox run --seccomp ./my-profile.json python app.py

# Clean up stopped containers created by sandbox:
sandbox prune
```

You can manage configuration with `sandbox config` or clean up stopped containers with `sandbox prune`. Use `sandbox --help` to see all available commands.

### Developing

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
- Rust (`cargo`, `rustc`)
- Ruby (`ruby`, `gem`)
- PHP (`php`, `composer`)
- Java (`java`, `javac`, `mvn`)

*(If a command doesn't match these, it falls back to a generic Alpine Linux image.)*

## Security & Isolation

Sandbox is designed to be "secure by default" when running untrusted code. Every container is hardened with:

- **Seccomp Security**: We block sensitive system calls like mount and ptrace to help prevent any accidental container escapes.
- **Read-Only Root**: The container's root filesystem is locked down, so only your project workspace and /tmp are writable.
- **Unprivileged Access**: All processes run as a standard user instead of root, adding another layer of safety.
- **Resource Management**: We limit memory, CPU, and process usage to ensure your system stays stable and avoids exhaustion.
- **Risk Mitigation**: High-risk system capabilities are disabled to keep the environment restricted.

## Configuration

Sandbox is highly configurable via two levels of configuration:

1. **Global Configuration** (`~/.sandbox/config.yaml`): Configures global container limits, default base images, and environment variable allowlists/blocklists.
2. **Workspace Configuration** (`.sandbox.yml`): Provides project-level customization such as pre-run dependency setup scripts.

For detailed information on configuring Sandbox, please see the [Configuration Guide](docs/configuration.md).

## License

MIT License. See [LICENSE](LICENSE) for details.
