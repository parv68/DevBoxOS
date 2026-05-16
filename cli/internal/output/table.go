package output

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/devboxos/devboxos/shared/types"
	pb "github.com/devboxos/devboxos/engine/proto"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	tableHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
)

// Success prints a success message.
func Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "%s %s\n", successStyle.Render("✓"), msg)
}

// Error prints an error message.
func Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", errorStyle.Render("✗"), msg)
}

// Warning prints a warning message.
func Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "%s %s\n", warnStyle.Render("⚠"), msg)
}

// Info prints an informational message.
func Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "%s %s\n", infoStyle.Render("ℹ"), msg)
}

// Dim prints a dimmed message.
func Dim(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "%s\n", dimStyle.Render(msg))
}

// Title prints a bold title.
func Title(text string) {
	fmt.Println(titleStyle.Render(text))
	fmt.Println(strings.Repeat("─", len(text)))
}

// Status prints the environment status as a table.
func Status(status *types.EnvironmentStatus) {
	Title("DevBoxOS Status")

	fmt.Printf("\nProject: %s\n", status.Path)
	fmt.Printf("Status:  %s\n\n", status.Status)

	if len(status.Services) == 0 {
		Dim("No services running")
		return
	}

	fmt.Printf("  %-15s %-12s %-10s %-8s %s\n",
		tableHeaderStyle.Render("SERVICE"),
		tableHeaderStyle.Render("STATUS"),
		tableHeaderStyle.Render("HEALTH"),
		tableHeaderStyle.Render("PORT"),
		tableHeaderStyle.Render("CONTAINER"),
	)

	for _, svc := range status.Services {
		containerID := svc.ContainerID
		if len(containerID) > 12 {
			containerID = containerID[:12]
		}

		fmt.Printf("  %-15s %-12s %-10s %-8d %s\n",
			svc.Name,
			svc.Status,
			svc.Health,
			svc.Port,
			containerID,
		)
	}
	fmt.Println()
}

// Doctor prints diagnostic results from the engine (proto type).
func DoctorProto(result *pb.DoctorResponse) {
	Title("DevBoxOS Doctor")

	if len(result.Issues) == 0 {
		Success("No issues detected")
		return
	}

	fmt.Println()
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "error":
			Error("%s", issue.Message)
			if issue.Details != "" {
				Dim("  %s", issue.Details)
			}
		case "warning":
			Warning("%s", issue.Message)
		case "info":
			Info("%s", issue.Message)
		default:
			fmt.Printf("  %s\n", issue.Message)
		}
	}

	if len(result.Suggestions) > 0 {
		fmt.Println()
		Title("Suggested Fixes")
		for _, suggestion := range result.Suggestions {
			fmt.Printf("  → %s\n", suggestion)
		}
	}
	fmt.Println()
}

// Doctor prints diagnostic results (legacy interface).
func Doctor(result interface{}) {
	if pb, ok := result.(*pb.DoctorResponse); ok {
		DoctorProto(pb)
		return
	}
	Dim("Diagnostics engine coming in Sprint 11-12")
	fmt.Println()
}

// Config prints configuration values.
func Config(cfg map[string]string) {
	Title("DevBoxOS Configuration")

	fmt.Println()
	for key, val := range cfg {
		fmt.Printf("  %-20s = %s\n", key, val)
	}
	fmt.Println()
}

// Spinner is a loading indicator.
type Spinner struct {
	message string
	done    chan bool
}

// NewSpinner creates a new spinner.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan bool),
	}
}

// Start starts the spinner.
func (s *Spinner) Start() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	go func() {
		for {
			select {
			case <-s.done:
				fmt.Printf("\r   \r")
				return
			default:
				fmt.Printf("\r %s %s", frames[i%len(frames)], s.message)
				i++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Update updates the spinner message.
func (s *Spinner) Update(message string) {
	s.message = message
}

// Stop stops the spinner.
func (s *Spinner) Stop() {
	s.done <- true
}
