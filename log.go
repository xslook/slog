package slog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func normalizeFilepath(dir, filename string) (string, error) {
	if dirInfo, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		if err = os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
	} else if !dirInfo.IsDir() {
		return "", fmt.Errorf("Filepath %s is not a valid directory", dir)
	}
	fs := filepath.Join(dir, filename)
	_, err := os.Stat(fs)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
	}
	return fs, nil
}

func initLogger(opts *Options) error {

	var outWriter zapcore.WriteSyncer
	if opts.Filename != "" {
		maxSize := opts.MaxSize
		if maxSize < 1 {
			maxSize = 1
		}
		maxBackups := opts.MaxBackups
		if maxBackups < 0 {
			maxBackups = 0
		}
		maxAge := opts.MaxAge
		if maxAge < 0 {
			maxAge = 0
		}
		if opts.Dir == "" {
			opts.Dir = "."
		}
	}

	var logLevel zapcore.Level
	switch strings.ToLower(opts.Level) {
	case "debug":
		logLevel = zapcore.DebugLevel
	case "info":
		logLevel = zapcore.InfoLevel
	case "warn":
		logLevel = zapcore.WarnLevel
	case "error":
		logLevel = zapcore.ErrorLevel
	case "fatal":
		logLevel = zapcore.FatalLevel
	default:
		logLevel = zapcore.InfoLevel
	}

	if opts.Stdout {
		if outWriter == nil {
			outWriter = zapcore.AddSync(os.Stdout)
		} else {
			outWriter = zapcore.NewMultiWriteSyncer(outWriter, os.Stdout)
		}
	}

	if outWriter == nil {
		return errors.New("No output writer")
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), outWriter, logLevel)
	samplerCore := zapcore.NewSampler(core, time.Second, 100, 100)
	logger = zap.New(samplerCore, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.DPanicLevel))
	zap.ReplaceGlobals(logger)

	return nil
}

// Options for logger
type Options struct {
	Dir       string
	Filename  string
	Level     string
	Rotate    bool // rotate log file or not
	LocalTime bool
	Stdout    bool

	FlushTick time.Duration

	MaxSize    int // The max size of single log file, default 200
	MaxBackups int // The max backup number of files
	MaxAge     int // The max keep days of log files
}

// Init a logger
func Init(opts *Options) error {
	return initLogger(opts)
}

func defaultOptions() *Options {
	dir := os.Getenv("LOG_DIR")
	if dir == "" {
		dir = "."
	}
	filename := os.Getenv("LOG_FILE")

	level := os.Getenv("LOG_LEVEL")
	stdout := filename == ""
	opts := &Options{
		Dir:        dir,
		Filename:   filename,
		Level:      level,
		LocalTime:  true,
		Stdout:     stdout,
		MaxSize:    200,
		MaxAge:     2,
		MaxBackups: 2,
	}
	return opts
}

// Logger ...
type Logger struct {
	zlog *zap.Logger
}

// New a logger
func New(opt *Options) *Logger {
	return nil
}

type ctxLoggerKey string

var ctxKey ctxLoggerKey = "logger"

// Mix create a new context wrap this logger
func (log *Logger) Mix(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey, log)
}

// From try extract logger instance from context
func From(ctx context.Context) *Logger {
	val := ctx.Value(ctxKey)
	if val == nil {
		return nil
	}
	log, ok := val.(*Logger)
	if !ok {
		return nil
	}
	return log
}

// Field ...
type Field = zap.Field

// With fields
func (log *Logger) With(fields ...Field) *Logger {
	log.zlog = log.zlog.With(fields...)
	return log
}

// GetLogger get a logger
func getLogger() *zap.Logger {
	if logger == nil {
		opts := defaultOptions()
		err := initLogger(opts)
		if err != nil {
			panic(err)
		}
	}
	return logger
}

// With zap fields
func With(fileds ...zap.Field) *zap.Logger {
	return getLogger().With(fileds...)
}

// Print log
func Print(args ...interface{}) {
	getLogger().Sugar().Info(args...)
}

// Printf log
func Printf(template string, args ...interface{}) {
	getLogger().Sugar().Infof(template, args...)
}

// Println log
func Println(args ...interface{}) {
	getLogger().Sugar().Info(args...)
}

// Fatal log
func Fatal(args ...interface{}) {
	getLogger().Sugar().Fatal(args...)
}

// Fatalf log
func Fatalf(template string, args ...interface{}) {
	getLogger().Sugar().Fatalf(template, args...)
}

// Fatalw log
func Fatalw(msg string, keysAndValues ...interface{}) {
	getLogger().Sugar().Fatalw(msg, keysAndValues...)
}

// Fatalln log
func Fatalln(args ...interface{}) {
	getLogger().Sugar().Fatal(args...)
}

// Panic log
func Panic(args ...interface{}) {
	getLogger().Sugar().Panic(args...)
}

// Panicf log
func Panicf(template string, args ...interface{}) {
	getLogger().Sugar().Panicf(template, args...)
}

// Panicw log
func Panicw(msg string, keysAndValues ...interface{}) {
	getLogger().Sugar().Panicw(msg, keysAndValues...)
}

// Panicln log
func Panicln(args ...interface{}) {
	getLogger().Sugar().Panic(args...)
}

// Debug log
func Debug(args ...interface{}) {
	getLogger().Sugar().Debug(args...)
}

// Debugf log
func Debugf(template string, args ...interface{}) {
	getLogger().Sugar().Debugf(template, args...)
}

// Debugw log
func Debugw(msg string, keysAndValues ...interface{}) {
	getLogger().Sugar().Debugw(msg, keysAndValues...)
}

// Info log
func Info(args ...interface{}) {
	getLogger().Sugar().Info(args...)
}

// Infof log
func Infof(template string, args ...interface{}) {
	getLogger().Sugar().Infof(template, args...)
}

// Infow log
func Infow(msg string, keysAndValues ...interface{}) {
	getLogger().Sugar().Infow(msg, keysAndValues...)
}

// Warn log
func Warn(args ...interface{}) {
	getLogger().Sugar().Warn(args...)
}

// Warnf log
func Warnf(template string, args ...interface{}) {
	getLogger().Sugar().Warnf(template, args...)
}

// Warnw log
func Warnw(msg string, keysAndValues ...interface{}) {
	getLogger().Sugar().Warnw(msg, keysAndValues...)
}

// Error log
func Error(args ...interface{}) {
	getLogger().Sugar().Error(args...)
}

// Errorf log
func Errorf(template string, args ...interface{}) {
	getLogger().Sugar().Errorf(template, args...)
}

// Errorw log
func Errorw(msg string, keysAndValues ...interface{}) {
	getLogger().Sugar().Errorw(msg, keysAndValues...)
}
