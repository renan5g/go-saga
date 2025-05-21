package saga

import (
	"io"
	"log"
	"os"
)

type StdLogger struct {
	logger *log.Logger
	prefix string
}

func NewStdLogger(prefix string) *StdLogger {
	return &StdLogger{
		logger: log.New(os.Stdout, prefix, log.LstdFlags),
		prefix: prefix,
	}
}

func (l *StdLogger) Printf(format string, v ...any) {
	l.logger.Printf(format, v...)
}

func (l *StdLogger) WithOutput(w io.Writer) *StdLogger {
	l.logger.SetOutput(w)
	return l
}

func (l *StdLogger) WithFlags(flags int) *StdLogger {
	l.logger.SetFlags(flags)
	return l
}

type NoOpLogger struct{}

func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

func (l *NoOpLogger) Printf(format string, v ...any) {}
