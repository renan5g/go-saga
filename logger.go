package saga

import "log"

// stdLogger adapta o log.Logger padrão para nossa interface Logger
type stdLogger struct {
	*log.Logger
}

func (l *stdLogger) Printf(format string, v ...any) {
	l.Logger.Printf(format, v...)
}

// NewStdLogger cria um logger padrão baseado no log.Logger do Go
func NewStdLogger(prefix string) Logger {
	return &stdLogger{
		Logger: log.New(log.Writer(), prefix, log.LstdFlags),
	}
}
