package regulog

import (
	"fmt"
	"io"
	"io/ioutil"
	"log/syslog"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	timeFormatStr = "Mon Jan 2 15:04:05"
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

// streamLogger implements the Logger interface and writes to the specified io.Writer ("stream").
// It mimics the stdlib logger including memory optimizations such as minimizing calls to fmt.Sprintf
// and using a shared buffer to format the message before writing it out.
type streamLogger struct {
	stream  io.Writer
	tag     string
	linebuf []byte
	lock    sync.Mutex
}

// If we ever want to print callers file:line info in the message.
func (w *streamLogger) callerFileLine() (string, int) {
	if _, file, line, ok := runtime.Caller(3); ok {
		return file, line
	}
	return "???", 0

}

func (w *streamLogger) output(timeStr, level, msg string) {
	// We need to serialize access to the linebuffer that is used to assemble the message \
	// as well as the output stream we will print to.
	w.lock.Lock()
	defer w.lock.Unlock()

	// appends a fixed width string 'str' into byte buffer 'b'. Appends spaces if 'str' is too short.
	fixedWidthStr := func(width int, str string, b []byte) []byte {
		// Write as many bytes as 'width', writing spaces if we run out of chars
		for i := 0; i < width; i++ {
			if i < len(str) {
				b = append(b, level[i])
			} else {
				b = append(b, ' ')
			}
		}
		return b
	}

	// save memory, (re)use a buffer instead of relying on fmt.Sprintf to format the output string
	w.linebuf = w.linebuf[:0]

	w.linebuf = append(w.linebuf, timeStr...)
	w.linebuf = append(w.linebuf, ' ')
	w.linebuf = append(w.linebuf, w.tag...)
	w.linebuf = append(w.linebuf, ' ')

	w.linebuf = append(w.linebuf, '[')
	w.linebuf = fixedWidthStr(5, level, w.linebuf)
	w.linebuf = append(w.linebuf, ']')

	w.linebuf = append(w.linebuf, ' ')
	w.linebuf = append(w.linebuf, msg...)

	w.stream.Write(w.linebuf)
}

func (w *streamLogger) Debugf(f string, v ...interface{}) {
	msg := fmt.Sprintf(f, v...)
	now := time.Now().Format(timeFormatStr)
	w.output(now, "DEBUG", msg)
}
func (w *streamLogger) Debugln(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	now := time.Now().Format(timeFormatStr)
	w.output(now, "DEBUG", msg)
}
func (w *streamLogger) Infof(f string, v ...interface{}) {
	msg := fmt.Sprintf(f, v...)
	now := time.Now().Format(timeFormatStr)
	w.output(now, "INFO", msg)
}
func (w *streamLogger) Infoln(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	now := time.Now().Format(timeFormatStr)
	w.output(now, "INFO", msg)
}
func (w *streamLogger) Warningf(f string, v ...interface{}) {
	msg := fmt.Sprintf(f, v...)
	now := time.Now().Format(timeFormatStr)
	w.output(now, "WARN", msg)
}
func (w *streamLogger) Warningln(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	now := time.Now().Format(timeFormatStr)
	w.output(now, "WARN", msg)
}
func (w *streamLogger) Errorf(f string, v ...interface{}) {
	msg := fmt.Sprintf(f, v...)
	now := time.Now().Format(timeFormatStr)
	w.output(now, "ERROR", msg)
}
func (w *streamLogger) Errorln(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	now := time.Now().Format(timeFormatStr)
	w.output(now, "ERROR", msg)
}

// New returns a Logger that logs to Stderr.
func New(name string) Logger {
	return &streamLogger{
		tag:     name,
		linebuf: []byte{},
		stream:  os.Stderr,
	}
}

func NewNull(name string) Logger {
	return &streamLogger{
		tag:     name,
		linebuf: []byte{},
		stream:  ioutil.Discard,
	}
}

// NewSyslog returns a Logger that logs to Syslog.
func NewSyslog(name string) Logger {
	l, err := syslog.New(syslog.LOG_DAEMON, name)
	if err != nil {
		panic(err)
	}
	return &wrappedSyslogWriter{*l}
}
