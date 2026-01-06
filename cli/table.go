package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// Table builds styled terminal tables.
type Table struct {
	headers []string
	rows    [][]string
	writer  io.Writer
}

// NewTable creates a new table with headers.
//
// Example:
//
//	t := cli.NewTable("ID", "Name", "Status")
//	t.AddRow("1", "Alice", "Active")
//	t.AddRow("2", "Bob", "Inactive")
//	t.Print()
func NewTable(headers ...string) *Table {
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
		writer:  os.Stdout,
	}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(values ...string) *Table {
	t.rows = append(t.rows, values)
	return t
}

// SetWriter sets the output writer.
func (t *Table) SetWriter(w io.Writer) *Table {
	t.writer = w
	return t
}

// Print renders the table to the configured writer.
func (t *Table) Print() {
	if len(t.rows) == 0 {
		return
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().
		Padding(0, 1)

	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("8"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}
			return cellStyle
		}).
		Headers(t.headers...).
		Rows(t.rows...)

	fmt.Fprintln(t.writer, tbl)
}

// String returns the table as a string.
func (t *Table) String() string {
	var sb strings.Builder
	t.SetWriter(&sb)
	t.Print()
	return sb.String()
}

// SimpleTable prints a quick table without building.
//
// Example:
//
//	cli.SimpleTable(
//	    []string{"Name", "Value"},
//	    [][]string{
//	        {"Host", "localhost"},
//	        {"Port", "8080"},
//	    },
//	)
func SimpleTable(headers []string, rows [][]string) {
	t := NewTable(headers...)
	for _, row := range rows {
		t.AddRow(row...)
	}
	t.Print()
}

// KeyValue prints a simple key-value list.
//
// Example:
//
//	cli.KeyValue(map[string]string{
//	    "Host": "localhost",
//	    "Port": "8080",
//	})
func KeyValue(data map[string]string) {
	keyStyle := lipgloss.NewStyle().Bold(true).Width(20)
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))

	for k, v := range data {
		fmt.Printf("%s %s\n", keyStyle.Render(k+":"), valStyle.Render(v))
	}
}

// List prints a bulleted list.
//
// Example:
//
//	cli.List("Item 1", "Item 2", "Item 3")
func List(items ...string) {
	bullet := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("•")
	for _, item := range items {
		fmt.Printf("  %s %s\n", bullet, item)
	}
}

// NumberedList prints a numbered list.
//
// Example:
//
//	cli.NumberedList("First", "Second", "Third")
func NumberedList(items ...string) {
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Width(4)
	for i, item := range items {
		fmt.Printf("%s %s\n", numStyle.Render(fmt.Sprintf("%d.", i+1)), item)
	}
}
