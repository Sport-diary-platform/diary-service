package logger

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/rs/zerolog"
	"golang.org/x/text/encoding/charmap"
)

// Interface -.
type Interface interface {
	Debug(message any, args ...any)
	Info(message any, args ...any)
	Warn(message any, args ...any)
	Error(message any, args ...any)
	Fatal(message any, args ...any)
	ErrorDetails(where, function, cause string, err error)
	FatalDetails(where, function, cause string, err error)
}

// Logger -.
type Logger struct {
	logger *zerolog.Logger
}

var _ Interface = (*Logger)(nil)

// New -.
func New(level string) *Logger {
	return NewWithWriter(level, os.Stdout)
}

func NewWithWriter(level string, writer io.Writer) *Logger {
	var l zerolog.Level

	switch strings.ToLower(level) {
	case "error":
		l = zerolog.ErrorLevel
	case "warn":
		l = zerolog.WarnLevel
	case "info":
		l = zerolog.InfoLevel
	case "debug":
		l = zerolog.DebugLevel
	default:
		l = zerolog.InfoLevel
	}

	// Устанавливаем уровень напрямую для этого экземпляра логгера
	logger := zerolog.New(writer).Level(l).With().Timestamp().Logger()

	return &Logger{
		logger: &logger,
	}
}

// Debug -.
func (l *Logger) Debug(message any, args ...any) {
	l.log(l.logger.Debug(), message, args...)
}

// Info -.
func (l *Logger) Info(message any, args ...any) {
	l.log(l.logger.Info(), message, args...)
}

// Warn -.
func (l *Logger) Warn(message any, args ...any) {
	l.log(l.logger.Warn(), message, args...)
}

// Error -.
func (l *Logger) Error(message any, args ...any) {
	l.log(l.logger.Error(), message, args...)
}

// Fatal -.
func (l *Logger) Fatal(message any, args ...any) {
	// zerolog.Fatal() сам вызовет os.Exit(1) после записи лога
	l.log(l.logger.Fatal(), message, args...)
}

func (l *Logger) ErrorDetails(where, function, cause string, err error) {
	l.details(l.logger.Error(), where, function, cause, err)
}

func (l *Logger) FatalDetails(where, function, cause string, err error) {
	l.details(l.logger.Fatal(), where, function, cause, err)
}

func (l *Logger) log(event *zerolog.Event, message any, args ...any) {
	if !event.Enabled() {
		return
	}

	var msg string
	switch m := message.(type) {
	case error:
		msg = m.Error()
	case string:
		msg = m
	default:
		msg = fmt.Sprintf("%v", m)
	}

	if len(args) != 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	event.Msg(normalizeEncoding(msg))
}

func (l *Logger) details(event *zerolog.Event, where, function, cause string, err error) {
	if !event.Enabled() {
		return
	}
	message := ""
	if err != nil {
		message = normalizeEncoding(err.Error())
	}
	formatted := normalizeEncoding(where + " - " + function + " - " + cause + " - " + message)
	event.
		Str("where", where).
		Str("function", function).
		Str("cause", cause).
		Err(errors.New(message)).
		Msg(formatted)
}

func normalizeEncoding(value string) string {
	if utf8.ValidString(value) {
		return value
	}
	decoded, err := charmap.Windows1251.NewDecoder().String(value)
	if err == nil && utf8.ValidString(decoded) {
		return decoded
	}
	return strings.ToValidUTF8(value, "�")
}
