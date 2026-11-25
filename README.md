# Golang System Monitor 1.0

A terminal-based system monitor for Windows, written in Go.

## Run on Windows

On Windows, use Go 1.27.4 or later, as specified in `go.mod`. Run the program from the project directory.

```powershell
go run .
```

To build and run an executable in PowerShell, use these commands.

```powershell
go build -o gsm.exe .
.\gsm.exe
```

## Commands

Type `gsm help` at the `>>` prompt to list commands. Live views refresh in place about once a second. Press Enter to stop a view and keep its command and last snapshot above a new prompt. Type `gsm exit` or press Ctrl+C to quit. Redirected input or output switches each view to a single snapshot.

| View     | Metrics                                                                                                                                         | Top 10 processes                                         |
| -------- | ----------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| `gsm -c` | CPU model, per-logical-processor bars, utilization, estimated current speed, base speed, sockets, cores, and process, thread, and handle counts | `gsm -c proc` by CPU usage                               |
| `gsm -m` | RAM type, usage, capacity, committed and cached memory, kernel pools, speed, form factor, and hardware-reserved memory                          | `gsm -m proc` by RAM working set                         |
| `gsm -g` | GPU models, engine utilization, dedicated and shared GPU memory, temperature, and hardware-reserved memory                                      | `gsm -g proc` by busiest GPU engine                      |
| `gsm -d` | Volume names, disk models, free and used space, active time, transfer rate, response time, and read and write speeds                            | `gsm -d proc` by disk I/O                                |
| `gsm -w` | Adapter model, signal strength or link usage, send and receive rates, SSID, Wi-Fi standard, and IP addresses                                    | `gsm -w proc` by TCP and UDP traffic on Wi-Fi interfaces |

`gsm -p` is a shortcut for `gsm -m proc`.

## Windows metrics and permissions

- Disk and Wi-Fi process rankings require a 64-bit build and permission to run a Windows kernel trace. If tracing fails, the disk view falls back to a labeled ranking by total process I/O.
- Windows location permission may be required for SSID, connection type, and signal strength.
- Individual metrics depend on the hardware, driver, and counters Windows provides. Missing readings are shown as `unavailable` where applicable.

## Development

Run `go test ./...` to check the usage bars and the command-line interface.

- `main.go` handles commands and the prompt.
- `internal/monitor/` collects and formats system metrics.
- `internal/terminal/` manages live refresh in the Windows console.
- `tests/` contains the tests.

Released under the [MIT license](LICENSE).
