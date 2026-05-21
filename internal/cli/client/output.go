package client

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

// OutputMode controls the output format.
type OutputMode int

const (
	OutputTable OutputMode = iota
	OutputJSON
)

// Printer handles formatted output to a writer.
type Printer struct {
	Mode   OutputMode
	Writer io.Writer
}

// NewPrinter creates a new Printer with the given mode, writing to stdout.
func NewPrinter(jsonMode bool) *Printer {
	mode := OutputTable
	if jsonMode {
		mode = OutputJSON
	}
	return &Printer{
		Mode:   mode,
		Writer: os.Stdout,
	}
}

// PrintJSON outputs data as indented JSON.
func (p *Printer) PrintJSON(data interface{}) {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
		return
	}
	fmt.Fprintln(p.Writer, string(output))
}

// PrintTable prints a table with headers and rows using tabwriter.
func (p *Printer) PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(p.Writer, 0, 0, 2, ' ', 0)

	// Print headers
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, h)
	}
	fmt.Fprintln(w)

	// Print rows
	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, col)
		}
		fmt.Fprintln(w)
	}

	w.Flush()
}

// PrintKeyValue prints key-value pairs aligned.
func (p *Printer) PrintKeyValue(pairs []KeyValue) {
	w := tabwriter.NewWriter(p.Writer, 0, 0, 2, ' ', 0)
	for _, kv := range pairs {
		fmt.Fprintf(w, "  %s:\t%s\n", kv.Key, kv.Value)
	}
	w.Flush()
}

// KeyValue is a simple key-value pair for display.
type KeyValue struct {
	Key   string
	Value string
}

// FormatSize formats bytes into a human-readable string.
func FormatSize(bytes int64) string {
	if bytes <= 0 {
		return "-"
	}
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.0f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatDuration formats seconds into a human-readable duration string.
func FormatDuration(seconds int64) string {
	if seconds <= 0 {
		return "-"
	}
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	mins := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// TruncateID shortens an ID to the given max length.
func TruncateID(id string, maxLen int) string {
	if len(id) <= maxLen {
		return id
	}
	return id[:maxLen] + "..."
}

// FormatPercent formats a 0-100 percentage value.
func FormatPercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value)
}
