# Conduit CLI

Command-line interface for running a Psiphon Conduit node - a volunteer-run proxy that relays traffic for users in censored regions.

## Quick Start

```bash
# First time setup (clones required dependencies)
make setup

# Build
make build

# Run
./dist/conduit start --psiphon-config /path/to/psiphon_config.json
```

## Requirements

- **Go 1.24.x** (Go 1.25+ is not supported due to psiphon-tls compatibility)
- Psiphon network configuration file (JSON)

The Makefile will automatically install Go 1.24.3 if not present.

## Configuration

Conduit requires a Psiphon network configuration file containing connection parameters. See `psiphon_config.example.json` for the expected format.

Contact Psiphon (info@psiphon.ca) to obtain valid configuration values.

## Usage

```bash
# Start with default settings
conduit start --psiphon-config ./psiphon_config.json

# Customize limits
conduit start --psiphon-config ./psiphon_config.json --max-clients 500 --bandwidth 10

# Verbose output (info messages)
conduit start --psiphon-config ./psiphon_config.json -v

# Debug output (everything)
conduit start --psiphon-config ./psiphon_config.json -vv

# Enable Prometheus metrics on port 9090 (binds to all interfaces)
conduit start --psiphon-config ./psiphon_config.json --metrics-address 9090

# Bind metrics to localhost only (more secure)
conduit start --psiphon-config ./psiphon_config.json --metrics-address 127.0.0.1:9090

# Bind to specific IP and port
conduit start --psiphon-config ./psiphon_config.json --metrics-address 0.0.0.0:9090
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--psiphon-config, -c` | - | Path to Psiphon network configuration file |
| `--max-clients, -m` | 200 | Maximum concurrent clients (1-1000) |
| `--bandwidth, -b` | 5 | Bandwidth limit per peer in Mbps (1-40) |
| `--data-dir, -d` | `./data` | Directory for keys and state |
| `--metrics-address` | "" | Address for Prometheus metrics endpoint (format: `ip:port`, `:port`, or `port`; empty to disable) |
| `-v` | - | Verbose output (use `-vv` for debug) |

## Building

```bash
# Build for current platform
make build

# Build with embedded config (single-binary distribution)
make build-embedded PSIPHON_CONFIG=./psiphon_config.json

# Build for all platforms
make build-all

# Individual platform builds
make build-linux       # Linux amd64
make build-linux-arm   # Linux arm64
make build-darwin      # macOS Intel
make build-darwin-arm  # macOS Apple Silicon
make build-windows     # Windows amd64
```

Binaries are output to `dist/`.

## Docker

### Build with embedded config (recommended)

```bash
docker build -t conduit \
  --build-arg PSIPHON_CONFIG=psiphon_config.json \
  -f Dockerfile.embedded .
```

### Run with persistent data

**Important:** The Psiphon broker tracks proxy reputation by key. Always use a persistent volume to preserve your key across container restarts, otherwise you'll start with zero reputation and may not receive client connections.

```bash
# Using a named volume (recommended)
docker run -d --name conduit \
  -v conduit-data:/home/conduit/data \
  --restart unless-stopped \
  conduit

# Or using a host directory
mkdir -p /path/to/data && chown 1000:1000 /path/to/data
docker run -d --name conduit \
  -v /path/to/data:/home/conduit/data \
  --restart unless-stopped \
  conduit
```

### Build without embedded config

If you prefer to mount the config at runtime:

```bash
docker build -t conduit .

docker run -d --name conduit \
  -v conduit-data:/home/conduit/data \
  -v /path/to/psiphon_config.json:/config.json:ro \
  --restart unless-stopped \
  conduit start --psiphon-config /config.json
```

## Data Directory

Keys and state are stored in the data directory (default: `./data`):
- `conduit_key.json` - Node identity keypair (preserve this!)

The broker builds reputation for your proxy based on this key. If you lose it, you'll need to build reputation from scratch.

## Prometheus Metrics

Conduit can export Prometheus metrics for monitoring. Enable it with the `--metrics-address` flag:

```bash
# Bind to all interfaces on port 9090
conduit start --psiphon-config ./psiphon_config.json --metrics-address 9090

# Or bind to localhost only (recommended for security)
conduit start --psiphon-config ./psiphon_config.json --metrics-address 127.0.0.1:9090
```

The address format can be:
- `port` - Just a port number (binds to all interfaces, e.g., `:9090`)
- `:port` - Explicitly bind to all interfaces
- `ip:port` - Bind to a specific IP address (e.g., `127.0.0.1:9090` or `0.0.0.0:9090`)

Metrics will be available at `http://<address>/metrics`. The following metrics are exported:

- `conduit_connecting_clients` - Number of clients currently connecting (gauge)
- `conduit_connected_clients` - Number of clients currently connected (gauge)
- `conduit_is_live` - Whether the proxy is connected to the broker (1) or not (0) (gauge)
- `conduit_bytes_up_total` - Total bytes uploaded to clients (counter)
- `conduit_bytes_down_total` - Total bytes downloaded from clients (counter)
- `conduit_config_info` - Configuration information with labels `max_clients` and `bandwidth_mbps` (gauge)
- `conduit_uptime_seconds` - Service uptime in seconds (gauge)

These metrics update every 5 seconds and can be scraped by Prometheus or any compatible monitoring system.

## License

GNU General Public License v3.0
