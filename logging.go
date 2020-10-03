package squirssi

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// fileFormatter prints log messages formatted for output in files.
type fileFormatter struct{}

func (f *fileFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	lvl := ""
	switch entry.Level {
	case logrus.InfoLevel:
		lvl = " INFO"
	case logrus.DebugLevel:
		lvl = "DEBUG"
	case logrus.WarnLevel:
		lvl = " WARN"
	case logrus.ErrorLevel:
		lvl = "ERROR"
	case logrus.FatalLevel:
		lvl = "FATAL"
	case logrus.TraceLevel:
		lvl = "TRACE"
	case logrus.PanicLevel:
		lvl = "PANIC"
	}
	return []byte(fmt.Sprintf("[%s] %s -> %s\n", time.Now().Format("2006-01-02 15:04:05"), lvl, entry.Message)), nil
}

// statusFormatter prints log messages formatted for the StatusWindow.
type statusFormatter struct{
	levelPadding int
}

func (f *statusFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	lvl := ""
	switch entry.Level {
	case logrus.InfoLevel:
		lvl = "[ INFO](fg:blue)"
	case logrus.DebugLevel:
		lvl = "[DEBUG](fg:white,bg:blue)"
	case logrus.WarnLevel:
		lvl = "[ WARN](fg:yellow)"
	case logrus.ErrorLevel:
		lvl = "[ERROR](fg:red)"
	case logrus.FatalLevel:
		lvl = "[FATAL](fg:white,bg:red,mod:bold)"
	case logrus.TraceLevel:
		lvl = "[TRACE](fg:white,mod:bold)"
	case logrus.PanicLevel:
		lvl = "[PANIC](fg:white,bg:red,mod:bold)"
	}
	return []byte(fmt.Sprintf(" %s[│](fg:grey) \x030%s\x03", lvl, entry.Message)), nil
}

// logFileWriterHook ensures that log messages are written to some output.
// This hook writes messages to stdout until Start is called, at which point
// the hook switches to writing to stderr.
// Because log messages are routed to the StatusWindow, it's possible for them
// to get lost if there is a fatal error preventing startup or if a runtime
// panic occurs.
type logFileWriterHook struct {
	file *os.File
	fmtr logrus.Formatter

	started bool

	mu sync.RWMutex
}

func newLogFileWriterHook() *logFileWriterHook {
	return &logFileWriterHook{file: os.Stdout, fmtr: &fileFormatter{}}
}

func (h *logFileWriterHook) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()
	// switch to stderr once started
	h.file = os.Stderr
	h.started = true
}

func (h *logFileWriterHook) Fire(entry *logrus.Entry) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.started {
		return nil
	}
	fire := func(f *os.File) {
		line, err := h.fmtr.Format(entry)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to format log message:", err)
			return
		}
		if _, err = f.Write(line); err != nil {
			fmt.Fprintln(os.Stderr, "failed to write log message:", err)
		}
	}
	if h.started {
		// todo: this is dead code, but writing to stderr messes up rendering
		// todo: should this write to a real file instead? need to config such things
		go fire(h.file)
	} else {
		// only block writes if the hook hasn't started yet.
		// this is done assuming that log messages that occur before starting
		// are usually happening right before the application is about to exit,
		// so if we launch a goroutine we risk exiting the process before the
		// write can complete.
		fire(h.file)
	}
	return nil
}

func (h *logFileWriterHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}
