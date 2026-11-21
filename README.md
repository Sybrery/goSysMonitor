# Golang System Monitor

A terminal-based system monitor written in Go. The goal is to bring CPU, RAM, GPU, disk, and process information into one terminal interface.

## Project status

- [x] CPU details and usage bars for each logical processor
- [x] RAM usage bar and swap usage
- [ ] GPU 0 and GPU 1 monitoring
- [ ] Disk monitoring
- [ ] Running process list

## Run locally

Use Go 1.27.4 or later, as specified in `go.mod`. From the project directory, run:

```powershell
go run .
```

Enter one of the following commands at the `>>` prompt. `Ctrl+C` or `gsm exit` to quit.

| Command    | What it shows                                                                                                                                    |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| `gsm help` | Available commands                                                                                                                               |
| `gsm -c`   | CPU model, overall usage, usage per logical processor, physical cores, logical processor count, occupied sockets, and reported maximum frequency |
| `gsm -m`   | RAM usage bar, total and available RAM, and swap usage and capacity.                                                                             |
| `gsm exit`   | Quit                                                                                                                                           |

## Usage bars

`gsm -c` shows one bar per logical processor. `gsm -m` shows a bar for RAM usage.

```text
CPU 1: [####......] 42.3%
CPU 2: [#.........] 18.0%
In use: 9.0 GiB [#####.....] 59.0%
```
