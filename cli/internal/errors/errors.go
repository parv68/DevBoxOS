package errors

import (
	"fmt"
	"os"
	"strings"
)

type ErrorCode string

const (
	ErrDockerNotRunning    ErrorCode = "DOCKER_NOT_RUNNING"
	ErrPortInUse           ErrorCode = "PORT_IN_USE"
	ErrConfigNotFound      ErrorCode = "CONFIG_NOT_FOUND"
	ErrConfigInvalid       ErrorCode = "CONFIG_INVALID"
	ErrEngineNotRunning    ErrorCode = "ENGINE_NOT_RUNNING"
	ErrServiceNotFound     ErrorCode = "SERVICE_NOT_FOUND"
	ErrNetworkError        ErrorCode = "NETWORK_ERROR"
	ErrPermissionDenied    ErrorCode = "PERMISSION_DENIED"
	ErrDiskSpace           ErrorCode = "DISK_SPACE"
	ErrVersionMismatch     ErrorCode = "VERSION_MISMATCH"
	ErrUnknown             ErrorCode = "UNKNOWN"
)

type DevBoxError struct {
	Code    ErrorCode
	Message string
	Hint    string
	Cause   error
}

func (e *DevBoxError) Error() string {
	return e.Message
}

func (e *DevBoxError) Unwrap() error {
	return e.Cause
}

func New(code ErrorCode, message string) *DevBoxError {
	return &DevBoxError{Code: code, Message: message}
}

func Wrap(code ErrorCode, message string, cause error) *DevBoxError {
	return &DevBoxError{Code: code, Message: message, Cause: cause}
}

func WithHint(err *DevBoxError, hint string) *DevBoxError {
	err.Hint = hint
	return err
}

func PrettyPrint(err error) {
	var dbErr *DevBoxError
	if As(err, &dbErr) {
		printDevBoxError(dbErr)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func printDevBoxError(err *DevBoxError) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Message)

	if err.Hint != "" {
		fmt.Fprintf(os.Stderr, "\nHint: %s\n", err.Hint)
	}

	if err.Cause != nil {
		fmt.Fprintf(os.Stderr, "\nCaused by: %v\n", err.Cause)
	}

	fmt.Fprintf(os.Stderr, "\nError code: %s\n", err.Code)
	fmt.Fprintf(os.Stderr, "Need help? https://devbox.sh/docs/errors/%s\n", strings.ToLower(string(err.Code)))
}

func As(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	type asInterface interface {
		As(interface{}) bool
	}
	if x, ok := err.(asInterface); ok {
		return x.As(target)
	}
	return false
}

func Is(err error, code ErrorCode) bool {
	var dbErr *DevBoxError
	if As(err, &dbErr) {
		return dbErr.Code == code
	}
	return false
}

func DockerNotRunning(cause error) *DevBoxError {
	return WithHint(
		New(ErrDockerNotRunning, "Docker is not running"),
		"Start Docker Desktop or run 'sudo systemctl start docker'",
	)
}

func PortInUse(port int, process string) *DevBoxError {
	return WithHint(
		New(ErrPortInUse, fmt.Sprintf("Port %d is already in use by %s", port, process)),
		fmt.Sprintf("Stop the process using port %d or configure a different port", port),
	)
}

func ConfigNotFound(path string) *DevBoxError {
	return WithHint(
		New(ErrConfigNotFound, fmt.Sprintf("Configuration file not found: %s", path)),
		"Run 'devbox init' to create a new configuration",
	)
}

func EngineNotRunning() *DevBoxError {
	return WithHint(
		New(ErrEngineNotRunning, "DevBoxOS engine is not running"),
		"Run 'devbox start' to start the engine daemon",
	)
}
