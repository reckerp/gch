# gch - Smart Git Branch Checkout Tool

A powerful Git branch checkout tool that provides fast and intuitive branch switching with fuzzy matching capabilities.

## Features

- Fuzzy branch name matching
- Interactive branch selector
- Remote branch tracking

## Installation

- Using Homebrew:

```sh
brew tap reckerp/tap
brew install gch
```

- Using `go install`:

```bash
go install github.com/reckerp/gch@latest
```

## Usage

### Basic Usage

```bash
# Checkout a branch using partial name
gch prod            # Checkout branch containing 'prod'
gch 123             # Checkout branch containing '123'

# Create and checkout a new branch
gch -b feature      # Create and checkout new branch 'feature'
gch -b feat/user    # Create and checkout new branch 'feat/user'

# Force checkout (discard local changes)
gch -f prod         # Force checkout branch containing 'prod'
gch -b -f feature   # Force create and checkout new branch

# Show interactive branch selector
gch                 # List all branches for interactive selection
```

### Command Line Options

- `-b, --branch`: Create and checkout a new branch with the given name
- `-f, --force`: Force checkout, discarding any local changes
- `--debug`: Enable debug output for branch matching process

## Development

### Building

```bash
# Build the binary
make build

# Install the tool to $(HOME)/bin
make install
```

## License

MIT

