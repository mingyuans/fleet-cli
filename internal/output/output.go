package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Icons
const (
	IconInfo    = "▸"
	IconSuccess = "✓"
	IconWarning = "!"
	IconError   = "✗"
	IconSkip    = "–"
)

var isTTY bool

func init() {
	isTTY = term.IsTerminal(int(os.Stdout.Fd()))
}

// SetTTY overrides TTY detection (for testing).
func SetTTY(v bool) {
	isTTY = v
}

// IsTTY returns whether stdout is a terminal.
func IsTTY() bool {
	return isTTY
}

// ANSI color codes (exported for Table cell coloring; unexported aliases below)
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorGrey   = "\033[90m"
)

const (
	reset  = ColorReset
	bold   = ColorBold
	red    = ColorRed
	green  = ColorGreen
	yellow = ColorYellow
	grey   = ColorGrey
)

func colorize(color, text string) string {
	if !isTTY {
		return text
	}
	return color + text + reset
}

func Green(text string) string  { return colorize(green, text) }
func Yellow(text string) string { return colorize(yellow, text) }
func Red(text string) string    { return colorize(red, text) }
func Grey(text string) string   { return colorize(grey, text) }
func Bold(text string) string   { return colorize(bold, text) }

// Success prints a success line.
func Success(format string, args ...any) {
	fmt.Printf("  %s %s\n", Green(IconSuccess), fmt.Sprintf(format, args...))
}

// Warning prints a warning line.
func Warning(format string, args ...any) {
	fmt.Printf("  %s %s\n", Yellow(IconWarning), fmt.Sprintf(format, args...))
}

// Error prints an error line.
func Error(format string, args ...any) {
	fmt.Printf("  %s %s\n", Red(IconError), fmt.Sprintf(format, args...))
}

// Skip prints a skip line.
func Skip(format string, args ...any) {
	fmt.Printf("  %s %s\n", Grey(IconSkip), fmt.Sprintf(format, args...))
}

// --- Multi-line pending area ---
// Each active task gets its own line. Lines update in-place using
// \033[NA (cursor up) + \033[J (clear to end of screen).
// All screen writes are batched into a single WriteString call.
// Caller must hold a mutex for all Pending* calls.

type pendingEntry struct {
	id   string
	text string
}

var (
	pending    []pendingEntry // ordered active tasks
	linesDrawn int            // how many pending lines are on screen right now
)

// PendingSet adds or updates a pending line, then redraws the pending area.
func PendingSet(id, text string) {
	if !isTTY {
		return
	}
	found := false
	for i := range pending {
		if pending[i].id == id {
			pending[i].text = text
			found = true
			break
		}
	}
	if !found {
		pending = append(pending, pendingEntry{id, text})
	}
	drawPending()
}

// PendingRemove removes a task from tracking (no screen update).
func PendingRemove(id string) {
	for i := range pending {
		if pending[i].id == id {
			pending = append(pending[:i], pending[i+1:]...)
			return
		}
	}
}

// PendingFlush erases the pending area, calls fn to print permanent output,
// then redraws remaining pending lines. This is atomic from the screen's perspective.
func PendingFlush(fn func()) {
	erasePending()
	fn()
	drawPending()
}

// erasePending removes all pending lines from screen.
func erasePending() {
	if !isTTY || linesDrawn == 0 {
		return
	}
	os.Stdout.WriteString(fmt.Sprintf("\033[%dA\033[J", linesDrawn))
	linesDrawn = 0
}

// drawPending erases old pending lines and draws current ones in a single write.
func drawPending() {
	if !isTTY {
		return
	}
	var buf strings.Builder
	if linesDrawn > 0 {
		fmt.Fprintf(&buf, "\033[%dA\033[J", linesDrawn)
	}
	for _, e := range pending {
		fmt.Fprintf(&buf, "  %s%s%s %s%s%s\n", grey, IconInfo, reset, grey, e.text, reset)
	}
	if buf.Len() > 0 {
		os.Stdout.WriteString(buf.String())
	}
	linesDrawn = len(pending)
}

// Info prints an info line.
func Info(format string, args ...any) {
	fmt.Printf("  %s %s\n", Bold(IconInfo), fmt.Sprintf(format, args...))
}

// Header prints a bold header line.
func Header(format string, args ...any) {
	fmt.Printf("\n%s\n", Bold(fmt.Sprintf(format, args...)))
}

// Progress returns a progress prefix string like "[3/10]".
func Progress(current, total int) string {
	return fmt.Sprintf("[%d/%d]", current, total)
}

// --- Table Formatter ---

// Table formats and prints a table with column headers and data.
type Table struct {
	headers []string
	rows    [][]string
	colors  [][]string // parallel to rows, optional color per cell
}

// NewTable creates a new table with the given headers.
func NewTable(headers ...string) *Table {
	return &Table{headers: headers}
}

// AddRow adds a row to the table. Colors is optional per-cell color.
func (t *Table) AddRow(cells []string, cellColors []string) {
	t.rows = append(t.rows, cells)
	t.colors = append(t.colors, cellColors)
}

// Print renders the table to stdout.
func (t *Table) Print() {
	t.Fprint(os.Stdout)
}

// Fprint renders the table to a writer.
func (t *Table) Fprint(w io.Writer) {
	colWidths := make([]int, len(t.headers))
	for i, h := range t.headers {
		colWidths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Print headers
	for i, h := range t.headers {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprintf(w, "%-*s", colWidths[i], h)
	}
	fmt.Fprintln(w)

	// Print separator
	totalWidth := 0
	for i, cw := range colWidths {
		totalWidth += cw
		if i > 0 {
			totalWidth += 2
		}
	}
	fmt.Fprintln(w, strings.Repeat("─", totalWidth))

	// Print rows
	for ri, row := range t.rows {
		for ci, cell := range row {
			if ci > 0 {
				fmt.Fprint(w, "  ")
			}
			display := cell
			if ri < len(t.colors) && ci < len(t.colors[ri]) && t.colors[ri][ci] != "" {
				display = colorize(t.colors[ri][ci], cell)
			}
			// Pad based on raw cell length (not colorized length)
			padding := colWidths[ci] - len(cell)
			fmt.Fprint(w, display)
			if padding > 0 && ci < len(row)-1 {
				fmt.Fprint(w, strings.Repeat(" ", padding))
			}
		}
		fmt.Fprintln(w)
	}
}

// --- Summary Formatter ---

// SummaryCounts holds counts for a summary line.
type SummaryCounts map[string]int

// Summary prints a summary line with counts.
func Summary(counts SummaryCounts, order []string) {
	parts := make([]string, 0, len(order))
	for _, key := range order {
		if c, ok := counts[key]; ok && c > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c, key))
		}
	}
	if len(parts) > 0 {
		fmt.Printf("\n%s\n", Bold(strings.Join(parts, ", ")))
	}
}
