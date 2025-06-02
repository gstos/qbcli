# qbcli

`qbcli` is a command-line tool written in Go for managing [qBittorrent](https://www.qbittorrent.org/) settings via its Web UI API.
It supports authentication, querying, reading and modifying settings via command line interface.
The goal was to programmatically changing the listening port,
ideal for dynamic VPN configuration. 
However, the high-end side of the API was made to be easily extensible.
So, feel free to hack it. If you share back your additions (and fixes!), 
I would be happy to incorporate them in the code base.

## Features

- Authenticate with qBittorrent Web UI
- Query and update preferences
- Set or read the current listening port
- Cookie-based session handling with optional caching
- Docker-friendly build

## Installation

### Build from Source

```bash
git clone https://github.com/youruser/qbcli.git
cd qbcli
go build -o qbcli ./main.go
```

### Docker Build

```bash
docker build -t qbcli .
docker run --rm qbcli --help
```

## Usage

### Set Listening Port

```bash
qbcli setListeningPort 45678

```
This sets the `listen_port` preference using the qBittorrent Web API.


### Get Listening Port

```bash
qbcli getListeningPort
45678
```
This returns the `listen_port` preference using the qBitorrent Wev API.


### Get Preferences

```bash
qbcli getPreferences
```

### Other Things

Check the syntax for other functionalities that were implemented.
Feel free to submit PRs with new features.

```bash
qbcli --help
```


## Configuration

The CLI reads credentials from environment variables, config files, or command-line flags (depending on your implementation).

Example `.env`:
```env
QBCLI_HOST=http://localhost:8080
QBCLI_USERNAME=admin
QBCLI_PASSWORD=adminadmin
```

Notice that command line arguments override environment settings.

## Authentication

`qbcli` uses session cookies and caches them to avoid logging in on every request. Cookies are stored (if enabled) in a local cache directory.

## 🐳 Running Behind VPN (e.g., ProtonVPN + Gluetun)

1. Run qBittorrent behind a VPN container (e.g. [`gluetun`](https://github.com/qdm12/gluetun)).
2. Use `qbcli` from a sidecar container or via `docker exec`.
3. Automate port updates using `VPN_PORT_FORWARDING_UP_COMMAND=/bin/sh -c 'qbcli --retry --max-retries 0 --delay 30s --timeout 5m setListeningPort {{PORTS}}'

Please refer to `Dockerfile` and `docker-compose.yml` in `./docker/gluetun` for a working solution.
Don't forget to adjust `--delay` and `--timeout` for your needs.


## Development

This project uses:

- Go 1.21+
- Cobra for CLI
- `slog` for structured logging
- Cookie-based session management

To run locally:

```bash
go run ./main.go setListeningPort 45678
```

If you want to extend its functionalities,
refer to `internal/qb/client/api.go` for the high-end side of API implementation
and to `cmd/set_listening_port.go` and `cmd/get_listening_port.go` for CLI extensions.

Versioning and pre-compiled binaries to be implemented in this repository soon.

## License

MIT ©

## Disclaimers

This is a toy project I hacked together over a weekend to experiment with Go — 
specifically, to programmatically update the listening port in qBittorrent via its WebAPI.

One of the workarounds I found involved using wget with connection retry enabled. 
That, however, required either bypassing authentication or manually handling session cookies 
(i.e., parsing response headers to extract SID tokens).

I noticed that httpie supports session management but lacks retry functionality. 
I could have written some shell scripts to handle retries, 
but by that point, I was already convinced I needed to reinvent the wheel…

If I were to write a proper CLI tool, it had to:
•	Manage session cookies transparently and reliably
•	Recover from transient network failures
•	Implement a configurable retry strategy

Distinguishing transient network errors 
(like timeouts or connection refusals) 
turned out to be nontrivial. 
I ended up using some obscure syscall error codes 
I found on Stack Overflow — likely POSIX-specific. 
If you know a better way, I’d love to hear from you.

Contributions and new features are very welcome.

Finally, I neither endorse nor have thoroughly audited the tools mentioned above. 
Use them at your own risk and comply with your local regulations, 
especially regarding copyright.
