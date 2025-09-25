# kport - SSH Port Forwarder TUI

A terminal user interface (TUI) application for SSH port forwarding that reads from your local SSH config, allows you to select SSH connections, detects running ports on remote hosts, and forwards them to localhost.

## Features

- **SSH Config Integration**: Automatically reads from `~/.ssh/config`
- **Interactive Host Selection**: Choose from configured SSH hosts using arrow keys
- **Automatic Port Detection**: Scans remote host for listening ports using `netstat`, `ss`, or `lsof`
- **Manual Port Forwarding**: Option to manually specify remote ports
- **Real-time Port Forwarding**: Creates SSH tunnels similar to VSCode's remote SSH port forwarding
- **Clean TUI Interface**: Built with Bubble Tea for a smooth terminal experience

## Installation

```bash
go build -o kport
```

## Usage

1. **Run the application**:
   ```bash
   ./kport
   ```

2. **Select SSH Host**: Use arrow keys to navigate and press Enter to select an SSH host from your config

3. **Choose Port**: 
   - The app will automatically detect open ports on the remote host
   - Select a port to forward using arrow keys and Enter
   - Press 'm' for manual port entry

4. **Port Forwarding**: Once started, the app will show the local port that forwards to your remote port

## Controls

### Host Selection
- `↑/↓` or `j/k`: Navigate through SSH hosts
- `Enter`: Select host and detect ports
- `m`: Manual port forwarding for selected host
- `q`: Quit application

### Port Selection
- `↑/↓` or `j/k`: Navigate through detected ports
- `Enter`: Start port forwarding for selected port
- `m`: Switch to manual port entry
- `Esc`: Go back to host selection
- `q`: Quit application

### Manual Port Entry
- `0-9`: Enter port number
- `Backspace`: Delete last digit
- `Enter`: Start forwarding for entered port
- `Esc`: Go back to previous screen
- `q`: Quit application

### Active Forwarding
- `Esc`: Stop forwarding and return to host selection
- `q`: Quit application

## SSH Configuration

The application reads from your standard SSH config file at `~/.ssh/config`. Example configuration:

```
Host my-server
    HostName example.com
    User myuser
    Port 22
    IdentityFile ~/.ssh/id_rsa

Host dev-box
    HostName dev.example.com
    User developer
    Port 2222
```

## Authentication

The application supports:
- SSH key-based authentication (using IdentityFile from config)
- SSH agent authentication (if SSH_AUTH_SOCK is set)

## Requirements

- Go 1.19 or later
- SSH access to remote hosts
- SSH config file at `~/.ssh/config`

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `golang.org/x/crypto/ssh` - SSH client implementation
- `golang.org/x/crypto/ssh/agent` - SSH agent support

## How It Works

1. **Config Parsing**: Reads and parses your SSH config file to extract host information
2. **SSH Connection**: Establishes SSH connection using configured authentication methods
3. **Port Detection**: Runs commands like `netstat -tlnp` on the remote host to find listening ports
4. **Port Forwarding**: Creates local TCP listener that forwards connections through SSH tunnel
5. **Traffic Relay**: Copies data bidirectionally between local and remote connections

## Expected Behavior

When you select an SSH host:

1. **Connection Success**: Shows detected ports or "No ports detected" with option for manual entry
2. **Connection Failure**: Shows "Could not connect" message with option for manual port forwarding
3. **Timeout**: Connection attempts timeout after 5 seconds to avoid hanging

The application gracefully handles connection failures and allows you to:
- Go back to host selection with `Esc`
- Try manual port forwarding with `m`
- Quit with `q`

## Limitations

- Password authentication is not implemented (use SSH keys or agent)
- Host key verification uses `InsecureIgnoreHostKey` (should be improved for production use)
- Port detection requires `netstat`, `ss`, or `lsof` on the remote host
- Connection failures are expected for non-existent or unreachable hosts

## License

MIT License