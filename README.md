# DigitalOcean Firewall Allowlister

A Go service that automatically manages DigitalOcean firewall rules by allowing traffic from supported sources. The service can run as a daemon with scheduled updates or as a one-shot command.

## Features

- 🔥 **Automatic Firewall Management**: Updates DigitalOcean firewall rules automatically
- ☁️ **Cloudflare Integration**: Fetches and allows current Cloudflare IP ranges
- 📊 **Netdata Support**: Resolves and allows IPs for Netdata monitoring domains
- ⏰ **Flexible Scheduling**: Runs on configurable cron schedules
- 🔧 **Multiple Modes**: Daemon mode for continuous operation, one-shot for manual execution
- 🧪 **Dry-Run Support**: Test changes without modifying actual firewall rules
- 📝 **Structured Logging**: JSON logging with configurable levels
- ⚙️ **Flexible Configuration**: YAML files, environment variables, and CLI flags

## Installation

### Using Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/kholisrag/do-firewall-allowlister/releases):

```bash
# Linux (x86_64)
curl -L https://github.com/kholisrag/do-firewall-allowlister/releases/latest/download/do-firewall-allowlister_Linux_x86_64.tar.gz | tar xz

# macOS (x86_64)
curl -L https://github.com/kholisrag/do-firewall-allowlister/releases/latest/download/do-firewall-allowlister_Darwin_x86_64.tar.gz | tar xz

# Windows (x86_64)
curl -L https://github.com/kholisrag/do-firewall-allowlister/releases/latest/download/do-firewall-allowlister_Windows_x86_64.zip -o do-firewall-allowlister.zip
```

### Using Docker

Multi-architecture Docker images are available:

```bash
# Run directly
docker run --rm ghcr.io/kholisrag/do-firewall-allowlister:latest --help

# With configuration file
docker run --rm -v $(pwd)/config.yaml:/config.yaml ghcr.io/kholisrag/do-firewall-allowlister:latest validate --config /config.yaml
```

### Using Go Install

```bash
go install github.com/kholisrag/do-firewall-allowlister/cmd/do-firewall-allowlister@latest
```

### From Source

```bash
git clone https://github.com/kholisrag/do-firewall-allowlister.git
cd do-firewall-allowlister

# Using Task (recommended)
task build

# Or using Go directly
go build -o do-firewall-allowlister ./cmd/do-firewall-allowlister
```

## Configuration

The service uses a hierarchical configuration system with the following priority (highest to lowest):

1. **CLI Flags** (highest priority)
2. **Environment Variables**
3. **YAML Configuration File** (lowest priority)

### Configuration File

Create a `config.yaml` file:

```yaml
logLevel: INFO

cron:
  schedule: "0 0 * * *" # Daily at midnight
  timezone: "UTC"

digitalocean:
  api_key: "your-digitalocean-api-key"
  firewall_id: "your-firewall-id"
  inbound_rules:
    - port: 80
      protocol: tcp
    - port: 443
      protocol: tcp

netdata:
  domains:
    - "app.netdata.cloud"
    - "api.netdata.cloud"
    - "mqtt.netdata.cloud"

cloudflare:
  ips_url: "https://api.cloudflare.com/client/v4/ips"
```

### Environment Variables

All configuration options can be set via environment variables with the `FIREWALL_ALLOWLISTER_` prefix:

```bash
export FIREWALL_ALLOWLISTER_LOG_LEVEL=DEBUG
export FIREWALL_ALLOWLISTER_DIGITALOCEAN_API_KEY=your-api-key
export FIREWALL_ALLOWLISTER_DIGITALOCEAN_FIREWALL_ID=your-firewall-id
export FIREWALL_ALLOWLISTER_CRON_SCHEDULE="0 */6 * * *"
```

### CLI Flags

All configuration options can be overridden with global CLI flags that work with any command:

```bash
./do-firewall-allowlister daemon \
  --config config.yaml \
  --log-level DEBUG \
  --digitalocean.api-key your-api-key \
  --digitalocean.firewall-id your-firewall-id \
  --dry-run
```

Global flags are available for all commands and include:

- `--config, -c`: Path to configuration file
- `--log-level`: Logging level
- `--digitalocean.api-key`: DigitalOcean API key
- `--digitalocean.firewall-id`: DigitalOcean firewall ID
- `--cron.schedule`: Cron schedule expression
- `--cron.timezone`: Timezone for cron schedule
- `--cloudflare.ips-url`: Cloudflare IPs API URL

## Usage

### Daemon Mode

Run the service continuously with scheduled updates:

```bash
# Run with default config.yaml
./do-firewall-allowlister daemon

# Run with custom config and dry-run mode
./do-firewall-allowlister daemon --config /path/to/config.yaml --dry-run

# Run with environment variables
FIREWALL_ALLOWLISTER_DIGITALOCEAN_API_KEY=your-key \
./do-firewall-allowlister daemon --dry-run
```

### One-Shot Mode

Execute firewall updates once and exit:

```bash
# Run once with default config
./do-firewall-allowlister oneshot

# Run once with dry-run to see what would be changed
./do-firewall-allowlister oneshot --dry-run

# Run with custom configuration
./do-firewall-allowlister oneshot --config /path/to/config.yaml
```

### Configuration Validation

Validate your configuration and test connectivity:

```bash
# Validate configuration file
./do-firewall-allowlister validate --config config.yaml

# Validate with environment variables
FIREWALL_ALLOWLISTER_DIGITALOCEAN_API_KEY=your-key \
./do-firewall-allowlister validate
```

### Status Check

Check the status of external services:

```bash
# Get status in JSON format
./do-firewall-allowlister status --config config.yaml

# Get status in table format
./do-firewall-allowlister status --format table
```

### Version Information

Get detailed version and build information:

```bash
# Text format
./do-firewall-allowlister version

# JSON format
./do-firewall-allowlister version --output json
```

## How It Works

1. **IP Collection**: The service fetches current Cloudflare IP ranges from their API and resolves IP addresses for configured Netdata domains
2. **Firewall Update**: It updates the specified DigitalOcean firewall with inbound rules allowing traffic from these IPs on configured ports
3. **Scheduling**: In daemon mode, it runs on a configurable cron schedule to keep firewall rules up-to-date
4. **Safety**: Dry-run mode allows you to see what changes would be made without actually modifying firewall rules

## Configuration Options

| Option         | Environment Variable                            | CLI Flag                     | Description                                     |
| -------------- | ----------------------------------------------- | ---------------------------- | ----------------------------------------------- |
| Log Level      | `FIREWALL_ALLOWLISTER_LOG_LEVEL`                | `--log-level`                | Logging level (DEBUG, INFO, WARN, ERROR, FATAL) |
| Cron Schedule  | `FIREWALL_ALLOWLISTER_CRON_SCHEDULE`            | `--cron.schedule`            | Cron expression for scheduling                  |
| Timezone       | `FIREWALL_ALLOWLISTER_CRON_TIMEZONE`            | `--cron.timezone`            | Timezone for cron schedule                      |
| DO API Key     | `FIREWALL_ALLOWLISTER_DIGITALOCEAN_API_KEY`     | `--digitalocean.api-key`     | DigitalOcean API key                            |
| Firewall ID    | `FIREWALL_ALLOWLISTER_DIGITALOCEAN_FIREWALL_ID` | `--digitalocean.firewall-id` | DigitalOcean firewall ID                        |
| Cloudflare URL | `FIREWALL_ALLOWLISTER_CLOUDFLARE_IPS_URL`       | `--cloudflare.ips-url`       | Cloudflare IPs API endpoint                     |

## Examples

### Docker Deployment

The project includes a multi-stage Dockerfile using distroless images for security:

```bash
# Build the image
docker build -t do-firewall-allowlister .

# Run with configuration
docker run --rm -v $(pwd)/config.yaml:/config.yaml \
  do-firewall-allowlister daemon --config /config.yaml

# Run one-shot with dry-run
docker run --rm -v $(pwd)/config.yaml:/config.yaml \
  do-firewall-allowlister oneshot --config /config.yaml --dry-run
```

Or use the pre-built multi-architecture images:

```bash
# Pull and run
docker run --rm -v $(pwd)/config.yaml:/config.yaml \
  ghcr.io/kholisrag/do-firewall-allowlister:latest daemon --config /config.yaml
```

### Systemd Service

Create `/etc/systemd/system/do-firewall-allowlister.service`:

```ini
[Unit]
Description=DigitalOcean Firewall Allowlister
After=network.target

[Service]
Type=simple
User=firewall-allowlister
WorkingDirectory=/opt/do-firewall-allowlister
ExecStart=/opt/do-firewall-allowlister/do-firewall-allowlister daemon --config /etc/do-firewall-allowlister/config.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: do-firewall-allowlister
spec:
  replicas: 1
  selector:
    matchLabels:
      app: do-firewall-allowlister
  template:
    metadata:
      labels:
        app: do-firewall-allowlister
    spec:
      containers:
        - name: do-firewall-allowlister
          image: your-registry/do-firewall-allowlister:latest
          command: ["./do-firewall-allowlister", "daemon"]
          env:
            - name: FIREWALL_ALLOWLISTER_DIGITALOCEAN_API_KEY
              valueFrom:
                secretKeyRef:
                  name: do-api-secret
                  key: api-key
            - name: FIREWALL_ALLOWLISTER_DIGITALOCEAN_FIREWALL_ID
              value: "your-firewall-id"
            - name: FIREWALL_ALLOWLISTER_LOG_LEVEL
              value: "INFO"
```

## Development

### Prerequisites

- Go 1.21 or later
- [Task](https://taskfile.dev/) (recommended) or Make
- [GoReleaser](https://goreleaser.com/) (for releases)
- Docker (for container builds)
- DigitalOcean API token with firewall management permissions

### Quick Start

```bash
# Clone the repository
git clone https://github.com/kholisrag/do-firewall-allowlister.git
cd do-firewall-allowlister

# Install dependencies
task deps

# Run tests
task test:short

# Build for current platform
task build

# See all available tasks
task --list
```

### Available Tasks

```bash
# Development
task build              # Build binary for current platform
task build:all          # Build binaries for all platforms
task test               # Run all tests with coverage
task test:short         # Run short tests
task test:integration   # Run integration tests
task test:coverage      # Generate coverage report

# Code Quality
task fmt                # Format Go code
task lint               # Run linters
task check              # Run all checks (format, lint, test)

# Docker
task docker:build       # Build Docker image
task docker:run         # Run Docker container
task docker:push        # Push Docker image

# Release
task release:dry        # Dry run release with GoReleaser
task release            # Create release with GoReleaser

# Utilities
task clean              # Clean build artifacts
task dev                # Run in development mode
task validate           # Validate configuration
```

### Manual Commands

If you prefer not to use Task:

```bash
# Build for current platform
go build -o do-firewall-allowlister ./cmd/do-firewall-allowlister

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o do-firewall-allowlister-linux-amd64 ./cmd/do-firewall-allowlister
GOOS=darwin GOARCH=amd64 go build -o do-firewall-allowlister-darwin-amd64 ./cmd/do-firewall-allowlister

# Run tests
go test -short ./...
go test -race -coverprofile=coverage.out ./...

# Format and lint
go fmt ./...
go vet ./...
golangci-lint run

# Docker build
docker build -t do-firewall-allowlister .
```

## Security Considerations

- **API Key Security**: Store DigitalOcean API keys securely using environment variables or secret management systems
- **Firewall Access**: Ensure the API key has minimal required permissions (firewall read/write only)
- **Network Security**: The service makes outbound HTTPS requests to Cloudflare and DigitalOcean APIs
- **Logging**: Avoid logging sensitive information; API keys are not logged by default

## Troubleshooting

### Common Issues

1. **Invalid API Key**: Ensure your DigitalOcean API key has firewall management permissions
2. **Firewall Not Found**: Verify the firewall ID exists and is accessible with your API key
3. **DNS Resolution Failures**: Check network connectivity for Netdata domain resolution
4. **Cron Schedule Errors**: Validate cron expressions using online cron validators

### Debug Mode

Enable debug logging to troubleshoot issues:

```bash
./do-firewall-allowlister daemon --log-level DEBUG --dry-run
```

### Validation

Always test with dry-run mode first:

```bash
./do-firewall-allowlister oneshot --dry-run
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [DigitalOcean](https://www.digitalocean.com/) for their excellent API
- [Cloudflare](https://www.cloudflare.com/) for providing public IP ranges
- [Netdata](https://www.netdata.cloud/) for their monitoring platform
- The Go community for excellent libraries and tools
