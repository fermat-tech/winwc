# winwc

A Unix-like `wc` command for Windows, written in Go.

Count lines, words, bytes, and characters in files — with multi-file totals, UTF-8 aware character counting, and color output. All the `wc` behaviour you already know, working natively on Windows.

Rename the binary to anything you like (`wc`, `count`, etc.) — all usage and error messages derive from the executable name automatically.

## Installation

### go install (recommended)

Requires [Go](https://golang.org) 1.21+.

```powershell
go install github.com/fermat-tech/winwc@latest
```

The binary lands in `%USERPROFILE%\go\bin`, which should already be on your `PATH`.

### Build from source

```powershell
git clone https://github.com/fermat-tech/winwc.git
cd winwc
go build -o winwc.exe .
```

### Download

Grab the latest binary from [Releases](https://github.com/fermat-tech/winwc/releases).

## Usage

```
winwc [OPTIONS] [FILE...]
```

Reads from stdin if no FILE is given. Use `-` to explicitly read stdin alongside other files. Glob patterns are expanded automatically (Windows shells don't expand them).

With no options, prints **lines**, **words**, and **bytes** — equivalent to `-lwc`.

## Options

| Flag | Description |
|------|-------------|
| `-l` | Print line count |
| `-w` | Print word count |
| `-c` | Print byte count |
| `-m` | Print character count (UTF-8 aware) |
| `-L` | Print length of longest line (in bytes) |
| `--color` | Force color output on |
| `--no-color` | Force color output off |

Flags can be combined: `-lw`, `-lm`, `-lwcm`, etc.

## Color output

Color works on all Windows terminals including the old Command Prompt (cmd.exe).

| Method | Description |
|--------|-------------|
| `--color` | Force on |
| `--no-color` | Force off |
| `WINWC_COLOR=always\|never\|auto` | Persistent preference |
| `NO_COLOR=1` | Disable color ([no-color.org](https://no-color.org)) |
| Auto (default) | On when stdout is a terminal, off when piped |

## Examples

```powershell
# Count lines, words, and bytes (default)
winwc file.txt

# Count lines only
winwc -l *.go

# Count words and lines
winwc -lw file.txt

# UTF-8 character count
winwc -m unicode.txt

# Longest line length
winwc -L *.md

# Multiple files — shows per-file counts and a total row
winwc *.go

# Count lines from stdin
git log --oneline | winwc -l

# Mix stdin and files
winwc - file.txt
```

## License

MIT
