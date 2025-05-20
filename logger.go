package saga

import (
	"fmt"
	"io"
	"log"
	"os"
)

// StdLogger é uma implementação simples da interface Logger
type StdLogger struct {
	logger *log.Logger
	prefix string
}

// NewStdLogger cria um novo logger para stdout com prefixo
func NewStdLogger(prefix string) *StdLogger {
	return &StdLogger{
		logger: log.New(os.Stdout, prefix, log.LstdFlags),
		prefix: prefix,
	}
}

// NewFileLogger cria um novo logger para arquivo com prefixo
func NewFileLogger(filename, prefix string) (*StdLogger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	return &StdLogger{
		logger: log.New(file, prefix, log.LstdFlags),
		prefix: prefix,
	}, nil
}

// Printf implementa a interface Logger
func (l *StdLogger) Printf(format string, v ...any) {
	l.logger.Printf(format, v...)
}

// WithOutput configura um novo destino para a saída do logger
func (l *StdLogger) WithOutput(w io.Writer) *StdLogger {
	l.logger.SetOutput(w)
	return l
}

// WithFlags configura flags do logger
func (l *StdLogger) WithFlags(flags int) *StdLogger {
	l.logger.SetFlags(flags)
	return l
}

// NoOpLogger é um logger que não faz nada
type NoOpLogger struct{}

// NewNoOpLogger cria um logger que não registra nada
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

// Printf implementa a interface Logger sem fazer nada
func (l *NoOpLogger) Printf(format string, v ...any) {}
