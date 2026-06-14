package logger

type Logger interface {
	WithFields(fields map[string]interface{}) Logger
	Info(args ...interface{})
	Error(args ...interface{})
	Debug(args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Errorf(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}
