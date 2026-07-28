package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

// ---- program name ----

var progName string

func init() {
	progName = strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
}

// ---- output writer ----

var stdout io.Writer = colorable.NewColorableStdout()
var stderr io.Writer = colorable.NewColorableStderr()

// ---- color ----

var useColor bool

const (
	colorReset  = "\033[0m"
	colorNum    = "\033[1;36m" // bold cyan — counts
	colorFile   = "\033[1;34m" // bold blue — filename
	colorTotal  = "\033[1;33m" // bold yellow — total row
)

func resolveColor(flagColor, flagNoColor bool) {
	if flagNoColor {
		useColor = false
		return
	}
	if flagColor {
		useColor = true
		return
	}
	if _, set := os.LookupEnv("NO_COLOR"); set {
		useColor = false
		return
	}
	switch strings.ToLower(os.Getenv("WINWC_COLOR")) {
	case "always", "yes", "1", "true":
		useColor = true
		return
	case "never", "no", "0", "false":
		useColor = false
		return
	}
	useColor = isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

func col(code, s string) string {
	if !useColor {
		return s
	}
	return code + s + colorReset
}

// ---- options ----

type options struct {
	lines   bool // -l
	words   bool // -w
	bytes   bool // -c
	chars   bool // -m
	maxLine bool // -L
	// color flags
	flagColor   bool
	flagNoColor bool
}

// showDefaults returns true when no counting flag was specified — show lines/words/bytes.
func (o *options) showDefaults() bool {
	return !o.lines && !o.words && !o.bytes && !o.chars && !o.maxLine
}

// ---- counts ----

type counts struct {
	lines   int64
	words   int64
	bytes   int64
	chars   int64
	maxLine int64
}

func (a *counts) add(b counts) {
	a.lines += b.lines
	a.words += b.words
	a.bytes += b.bytes
	a.chars += b.chars
	if b.maxLine > a.maxLine {
		a.maxLine = b.maxLine
	}
}

// ---- ctrl-d as EOF ----

// ctrlDReader wraps an interactive stdin so a Ctrl-D byte (0x04) in the
// stream ends input, mirroring Unix terminal behavior. Windows consoles
// have no such convention — the native "end input" key is Ctrl-Z — and
// since this tool doesn't run the console in raw mode, a typed Ctrl-D
// arrives as a literal 0x04 byte once Enter is pressed rather than as an
// out-of-band signal, so it has to be detected in the byte stream here.
// Only wired up when stdin is a terminal (see stdinReader): piped or
// redirected input passes through unchanged, since 0x04 there is just data.
type ctrlDReader struct {
	r    io.Reader
	done bool
}

func (c *ctrlDReader) Read(p []byte) (int, error) {
	if c.done {
		return 0, io.EOF
	}
	n, err := c.r.Read(p)
	if i := bytes.IndexByte(p[:n], 0x04); i >= 0 {
		c.done = true
		return i, io.EOF
	}
	return n, err
}

// stdinReader returns os.Stdin, wrapped with Ctrl-D-as-EOF handling when
// stdin is an interactive terminal.
func stdinReader() io.Reader {
	if isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return &ctrlDReader{r: os.Stdin}
	}
	return os.Stdin
}

// ---- counting ----

// countReader counts lines, words, bytes, chars, and the longest line directly
// from the raw byte/rune stream. Counting must not be done via bufio.Scanner:
// it strips line terminators, so bytes/chars can only be reconstructed by adding
// a fixed +1 per line — which miscounts a final line with no trailing newline
// (e.g. `printf hi` -> 3 bytes instead of 2) and CRLF endings. Matching GNU wc:
// lines is the number of '\n' bytes, bytes/chars count the actual input, and a
// word is a maximal run of non-whitespace.
func countReader(r io.Reader) (counts, error) {
	var c counts
	br := bufio.NewReaderSize(r, 64*1024)
	var curLine int64 // bytes on the current line, excluding the newline
	inWord := false
	for {
		ru, size, err := br.ReadRune()
		if size > 0 {
			c.bytes += int64(size)
			c.chars++
			if unicode.IsSpace(ru) {
				inWord = false
			} else if !inWord {
				inWord = true
				c.words++
			}
			if ru == '\n' {
				c.lines++
				if curLine > c.maxLine {
					c.maxLine = curLine
				}
				curLine = 0
			} else {
				curLine += int64(size)
			}
		}
		if err != nil {
			if err == io.EOF {
				if curLine > c.maxLine {
					c.maxLine = curLine
				}
				return c, nil
			}
			return c, err
		}
	}
}

// ---- formatting ----

// fieldWidth computes a consistent column width across all counts being shown.
func fieldWidth(totals counts, opts *options) int {
	max := int64(0)
	if opts.showDefaults() || opts.lines {
		if totals.lines > max {
			max = totals.lines
		}
	}
	if opts.showDefaults() || opts.words {
		if totals.words > max {
			max = totals.words
		}
	}
	if opts.showDefaults() || opts.bytes {
		if totals.bytes > max {
			max = totals.bytes
		}
	}
	if opts.chars {
		if totals.chars > max {
			max = totals.chars
		}
	}
	if opts.maxLine {
		if totals.maxLine > max {
			max = totals.maxLine
		}
	}
	w := len(fmt.Sprintf("%d", max))
	if w < 1 {
		w = 1
	}
	return w
}

func printCounts(c counts, name string, w int, opts *options, nameColor string) {
	var fields []string

	numf := func(n int64) string {
		return col(colorNum, fmt.Sprintf("%*d", w, n))
	}

	if opts.showDefaults() || opts.lines {
		fields = append(fields, numf(c.lines))
	}
	if opts.showDefaults() || opts.words {
		fields = append(fields, numf(c.words))
	}
	if opts.showDefaults() || opts.bytes {
		fields = append(fields, numf(c.bytes))
	}
	if opts.chars {
		fields = append(fields, numf(c.chars))
	}
	if opts.maxLine {
		fields = append(fields, numf(c.maxLine))
	}

	line := strings.Join(fields, " ")
	if name != "" {
		line += " " + col(nameColor, name)
	}
	fmt.Fprintln(stdout, line)
}

// ---- fatal / warn ----

func fatal(format string, args ...any) {
	fmt.Fprintf(stderr, progName+": "+format+"\n", args...)
	os.Exit(1)
}

func warn(format string, args ...any) {
	fmt.Fprintf(stderr, progName+": "+format+"\n", args...)
}

// ---- glob expansion ----

func expandGlobs(paths []string) []string {
	var out []string
	for _, p := range paths {
		if !strings.ContainsAny(p, "*?[") {
			out = append(out, p)
			continue
		}
		matches, err := filepath.Glob(p)
		if err != nil || len(matches) == 0 {
			out = append(out, p)
			continue
		}
		out = append(out, matches...)
	}
	return out
}

// ---- flag parsing ----

func parseFlags(args []string) (*options, []string) {
	opts := &options{}
	var rest []string

	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		switch arg {
		case "--color":
			opts.flagColor = true
			i++
			continue
		case "--no-color":
			opts.flagNoColor = true
			i++
			continue
		case "--help", "-h":
			usage()
		}
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			for _, ch := range arg[1:] {
				switch ch {
				case 'l':
					opts.lines = true
				case 'w':
					opts.words = true
				case 'c':
					opts.bytes = true
				case 'm':
					opts.chars = true
				case 'L':
					opts.maxLine = true
				default:
					fatal("unknown option -%c", ch)
				}
			}
			i++
			continue
		}
		rest = append(rest, arg)
		i++
	}
	return opts, rest
}

// ---- usage ----

func usage() {
	fmt.Fprintf(stderr, `Usage: %s [OPTIONS] [FILE...]

Count lines, words, and bytes in each FILE, and total if more than one FILE.
Reads from stdin if no FILE is given (use - to explicitly read stdin).

Options:
  -l          Print line count
  -w          Print word count
  -c          Print byte count
  -m          Print character count (UTF-8 aware)
  -L          Print length of longest line
  --color     Force color output on
  --no-color  Force color output off

With no options, prints lines, words, and bytes (equivalent to -lwc).

Color control (lowest to highest priority):
  Auto          Enabled when stdout is a terminal
  WINWC_COLOR   Set to always, never, or auto
  NO_COLOR      Any value disables color (https://no-color.org)
  --color       Force enable
  --no-color    Force disable

Examples:
  %s file.txt
  %s -l *.go
  %s -lw file.txt
  %s -m unicode.txt
  %s -L *.md
  git log --oneline | %s -l
`, progName, progName, progName, progName, progName, progName, progName)
	os.Exit(0)
}

// ---- main ----

func main() {
	opts, rest := parseFlags(os.Args[1:])
	resolveColor(opts.flagColor, opts.flagNoColor)

	files := expandGlobs(rest)

	var total counts
	var exitCode int

	if len(files) == 0 {
		// Read from stdin
		c, err := countReader(stdinReader())
		if err != nil {
			warn("stdin: %v", err)
			exitCode = 1
		}
		w := fieldWidth(c, opts)
		printCounts(c, "", w, opts, colorFile)
		os.Exit(exitCode)
	}

	// Collect all counts first so we can compute a consistent field width
	type result struct {
		name string
		c    counts
		err  error
	}
	var results []result

	for _, path := range files {
		if path == "-" {
			c, err := countReader(stdinReader())
			results = append(results, result{"-", c, err})
			total.add(c)
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			warn("cannot open %q: %v", path, err)
			exitCode = 1
			results = append(results, result{path, counts{}, err})
			continue
		}
		c, err := countReader(f)
		f.Close()
		if err != nil {
			warn("%s: %v", path, err)
			exitCode = 1
		}
		results = append(results, result{path, c, err})
		total.add(c)
	}

	showTotal := len(files) > 1
	ref := total
	if !showTotal {
		ref = results[0].c
	}
	w := fieldWidth(ref, opts)

	for _, r := range results {
		if r.err != nil && r.name != "-" {
			continue
		}
		printCounts(r.c, r.name, w, opts, colorFile)
	}

	if showTotal {
		printCounts(total, "total", w, opts, colorTotal)
	}

	os.Exit(exitCode)
}
