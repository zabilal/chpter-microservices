package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap logger
type Logger struct {
	*zap.Logger
}

// Config holds logger configuration
type Config struct {
	Level      string
	OutputPath string
	Encoding   string
}

// NewLogger creates a new logger instance
func NewLogger(level string) *Logger {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(getLogLevel(level))
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.DisableStacktrace = true

	logger, err := config.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.FatalLevel),
	)
	if err != nil {
		panic(err)
	}

	return &Logger{logger}
}

// NewDevelopmentLogger creates a new development logger instance
func NewDevelopmentLogger() *Logger {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.DisableStacktrace = true

	logger, err := config.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.FatalLevel),
	)
	if err != nil {
		panic(err)
	}

	return &Logger{logger}
}

// NewCustomLogger creates a new logger with custom configuration
func NewCustomLogger(cfg *Config) *Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg.Encoding == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	outputPath := cfg.OutputPath
	if outputPath == "" {
		outputPath = "stdout"
	}

	var sink zapcore.WriteSyncer
	if outputPath == "stdout" {
		sink = zapcore.AddSync(os.Stdout)
	} else {
		file, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		sink = zapcore.AddSync(file)
	}

	core := zapcore.NewCore(
		encoder,
		sink,
		getLogLevel(cfg.Level),
	)

	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return &Logger{logger}
}

func getLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// WithFields adds structured context to the logger
func (l *Logger) WithFields(fields ...zapcore.Field) *Logger {
	return &Logger{l.With(fields...)}
}

// WithError adds an error field to the logger
func (l *Logger) WithError(err error) *Logger {
	return &Logger{l.With(zap.Error(err))}
}

// WithService adds a service field to the logger
func (l *Logger) WithService(service string) *Logger {
	return &Logger{l.With(zap.String("service", service))}
}
