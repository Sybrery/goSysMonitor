# Golang System Monitor

A terminal-based system monitor for Windows, written in Go.

## Project status

- [x] CPU details and usage bars for each logical processor
- [x] RAM usage, commit charge, system cache, kernel memory pools, and hardware-reserved RAM
- [ ] GPU 0 and GPU 1 monitoring
- [x] Disk monitoring
- [ ] Running process list

## Run locally

On Windows, use Go 1.27.4 or later, as specified in `go.mod`. Run the program from the project directory.

```powershell
go run .
```

Enter one of the following commands at the `>>` prompt. `Ctrl+C` or `gsm exit` to quit.

| Command    | What it shows |
| ---------- | ------------- |
| `gsm help` | Command list |
| `gsm -c`   | CPU model, overall usage, usage per logical processor, current speed, physical cores, logical processor count, occupied sockets, and reported reference frequency |
| `gsm -m`   | Memory type, RAM usage and capacity, committed memory, system cache, kernel memory pools, speed, form factor, and hardware-reserved RAM |
| `gsm -g`   | Placeholder for GPU monitoring |
| `gsm -d`   | Volume names and disk models, free and used space, active time, transfer rate, average response time, and read/write speeds |
| `gsm -p`   | Placeholder for the process list |
| `gsm exit` | Quit |

## Usage bars

`gsm -c` shows one bar per logical processor. `gsm -m` shows a bar for RAM usage.

```text
CPU 1: [####......] 42.3%
CPU 2: [#.........] 18.0%
In use: 9.0 GiB [#####.....] (59.0%)
```
