package utils

import (
	"fmt"
	"io"

	tm "github.com/buger/goterm"
	"github.com/fatih/color"
	"github.com/janeczku/go-spinner"
)

// TerminalTable represents a a terminal table to output.
type TerminalTable struct {
	Table *tm.Table
}

func (t *TerminalTable) AddRow(pattern string, args ...interface{}) {
	fmt.Fprintf(t.Table, pattern, args...)
}

func (t *TerminalTable) Print() {
	tm.Print(t.Table)
	tm.Flush()
}

// CreateTerminalTable creates a new terminal table.
func CreateTerminalTable() TerminalTable {
	return TerminalTable{
		Table: tm.NewTable(0, 10, 5, ' ', 0),
	}
}

// ClearLine clears the last line in the terminal.
func ClearLine() {
	fmt.Printf("\033[2K")
	fmt.Println()
	fmt.Printf("\033[1A")
}

type TerminalSpinnerLogger struct{}

func (t *TerminalSpinnerLogger) Write(msg []byte) (int, error) {
	fmt.Printf("\033[2K")
	fmt.Println()
	fmt.Printf("\033[1A")
	fmt.Printf("%s", string(msg))
	return len(msg), nil
}

type TerminalSpinner struct {
	SpinnerText   string
	CompletedText string
	FailureText   string
	Element       *spinner.Spinner
}

func (t *TerminalSpinner) Create() {
	t.Element = spinner.StartNew(t.SpinnerText)
}

func (t *TerminalSpinner) SetOutput(writer io.Writer) {
	t.Element.Output = writer
}

func (t *TerminalSpinner) Stop() {
	t.Element.Stop()
	Print(t.CompletedText)
}

func (t *TerminalSpinner) Fail() {
	t.Element.Stop()
	Print(t.FailureText)
}

func (t *TerminalSpinner) FailWithMessage(message string, err error) error {
	t.Fail()
	return NewErrorMessage(message, err)
}

// CreateNewTerminalSpinner creates a new terminal spinner.
func CreateNewTerminalSpinner(spinnerText, completedText, failureText string) TerminalSpinner {
	spinner := TerminalSpinner{
		SpinnerText:   spinnerText,
		CompletedText: fmt.Sprintf("✅ %s", completedText),
		FailureText:   fmt.Sprintf("❌ %s", failureText),
	}
	spinner.Create()
	return spinner
}

func TextColor(text string, textColor ...color.Attribute) string {
	c := color.New(textColor...).SprintFunc()
	return c(text)
}

func Print(items ...interface{}) {
	fmt.Println(items...)
}

func Printf(pattern string, items ...interface{}) {
	fmt.Printf(pattern, items...)
}
