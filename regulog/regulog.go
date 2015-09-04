package regulog

import (
	"fmt"
	"log/syslog"
	"os"

	"github.com/apsdehal/go-logger"
)

// Logger defines a minimal logging interface.
// Note: All functions append a trailing newline if one doesn't exist.
type Logger interface {
	Debugf(format string, v ...interface{})
	Debugln(v ...interface{})

	Infof(format string, v ...interface{})
	Infoln(v ...interface{})

	Warningf(format string, v ...interface{})
	Warningln(v ...interface{})

	Errorf(format string, v ...interface{})
	Errorln(v ...interface{})
}

// wraps around a syslog.Writer to make it adhere to the `Logger` interface.
type wrappedSyslogWriter struct {
	syslog.Writer
}

func (w *wrappedSyslogWriter) Debugf(f string, v ...interface{}) {
	w.Writer.Debug(fmt.Sprintf(f, v...))
}
func (w *wrappedSyslogWriter) Debugln(v ...interface{}) {
	w.Writer.Debug(fmt.Sprintln(v...))
}
func (w *wrappedSyslogWriter) Infof(f string, v ...interface{}) {
	w.Writer.Info(fmt.Sprintf(f, v...))
}
func (w *wrappedSyslogWriter) Infoln(v ...interface{}) {
	w.Writer.Info(fmt.Sprintln(v...))
}
func (w *wrappedSyslogWriter) Warningf(f string, v ...interface{}) {
	w.Writer.Warning(fmt.Sprintf(f, v...))
}
func (w *wrappedSyslogWriter) Warningln(v ...interface{}) {
	w.Writer.Warning(fmt.Sprintln(v...))
}
func (w *wrappedSyslogWriter) Errorf(f string, v ...interface{}) {
	w.Writer.Err(fmt.Sprintf(f, v...))
}
func (w *wrappedSyslogWriter) Errorln(v ...interface{}) {
	w.Writer.Err(fmt.Sprintln(v...))
}

// wraps around apsdehal.Logger to  make it adhere to the `Logger` interface
type wrappedLogger struct {
	logger.Logger
}

func (w *wrappedLogger) Debugf(f string, v ...interface{}) {
	w.Logger.Debug(fmt.Sprintf(f, v...))
}
func (w *wrappedLogger) Debugln(v ...interface{}) {
	w.Logger.Debug(fmt.Sprintln(v...))
}
func (w *wrappedLogger) Infof(f string, v ...interface{}) {
	w.Logger.Info(fmt.Sprintf(f, v...))
}
func (w *wrappedLogger) Infoln(v ...interface{}) {
	w.Logger.Info(fmt.Sprintln(v...))
}
func (w *wrappedLogger) Warningf(f string, v ...interface{}) {
	w.Logger.Warning(fmt.Sprintf(f, v...))
}
func (w *wrappedLogger) Warningln(v ...interface{}) {
	w.Logger.Warning(fmt.Sprintln(v...))
}
func (w *wrappedLogger) Errorf(f string, v ...interface{}) {
	w.Logger.Error(fmt.Sprintf(f, v...))
}
func (w *wrappedLogger) Errorln(v ...interface{}) {
	w.Logger.Error(fmt.Sprintln(v...))
}

// New returns a Logger that logs to Stderr.
func New(name string) Logger {
	l, err := logger.New(name, 1, os.Stderr)
	if err != nil {
		panic(err)
	}
	return &wrappedLogger{*l}
}

// NewSyslog returns a Logger that logs to Syslog.
func NewSyslog(name string) Logger {
	l, err := syslog.New(syslog.LOG_DAEMON, name)
	if err != nil {
		panic(err)
	}
	return &wrappedSyslogWriter{*l}
}
