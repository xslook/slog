package zg

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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

func initLogger(opts *Option) (*zap.Logger, error) {

	var outWriter zapcore.WriteSyncer
	if opts.Filename != "" {
		if opts.Dir == "" {
			opts.Dir = "."
		}
		path := filepath.Join(opts.Dir, opts.Filename)
		info, err := os.Stat(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			fs, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return nil, err
			}
			defer fs.Close()
		}
		if info.IsDir() {
			return nil, nil
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
		return nil, errors.New("No output writer")
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), outWriter, logLevel)
	samplerCore := zapcore.NewSampler(core, time.Second, 100, 100)
	logger := zap.New(samplerCore, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.DPanicLevel))
	zap.ReplaceGlobals(logger)

	return logger, nil
}

// Option for logger
type Option struct {
	Dir       string
	Filename  string
	Level     string
	LocalTime bool
	Stdout    bool
}

// Init a logger
func Init(opts *Option) error {
	_, err := initLogger(opts)
	return err
}

func defaultOption() *Option {
	dir := os.Getenv("LOG_DIR")
	if dir == "" {
		dir = "."
	}
	filename := os.Getenv("LOG_FILE")

	level := os.Getenv("LOG_LEVEL")
	stdout := filename == ""
	opts := &Option{
		Dir:       dir,
		Filename:  filename,
		Level:     level,
		LocalTime: true,
		Stdout:    stdout,
	}
	return opts
}

// Logger ...
type Logger struct {
	core *zap.Logger
}

var gLogger = newNoOp()

func newNoOp() *Logger {
	return &Logger{
		core: zap.L(),
	}
}

func newTraceID() string {
	var bts [16]byte
	_, err := io.ReadFull(rand.Reader, bts[:])
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bts[:])
}

// New a logger
func New(opt *Option) (*Logger, error) {
	core, err := initLogger(opt)
	if err != nil {
		return nil, err
	}
	logger := &Logger{
		core: core,
	}
	return logger, nil
}

// clone a new logger
func (log *Logger) clone() *Logger {
	cp := *log
	return &cp
}

type traceType string

var (
	traceKey traceType
)

// Mix create a new context wrap this logger
func (log *Logger) Mix(ctx context.Context) context.Context {
	id := newTraceID()
	return context.WithValue(ctx, traceKey, id)
}

// Trace create a new context mixed with logger
func Trace(ctx context.Context) context.Context {
	id := newTraceID()
	return context.WithValue(ctx, traceKey, id)
}

const (
	dftTraceKey   = "_s"
	dftLatencyKey = "_t"
)

// In try extract logger instance from context
func In(ctx context.Context) *Logger {
	val := ctx.Value(traceKey)
	if val == nil {
		return gLogger.With(String(dftTraceKey, newTraceID()))
	}
	v, ok := val.(string)
	if !ok {
		return gLogger.With(String(dftTraceKey, newTraceID()))
	}
	return gLogger.With(String(dftTraceKey, v))
}

// With fields
func (log *Logger) With(fields ...Field) *Logger {
	l := log.clone()
	l.core = log.core.With(fields...)
	return l
}

// Named create a named logger
func (log *Logger) Named(name string) *Logger {
	l := log.clone()
	l.core = log.core.Named(name)
	return l
}

// Debug log
func (log *Logger) Debug(msg string) {
	log.core.Debug(msg)
}

// Info log
func (log *Logger) Info(msg string) {
	log.core.Info(msg)
}

func (log *Logger) Infof(template string, args ...interface{}) {
	log.core.Sugar().Infof(template, args...)
}

// Warn log
func (log *Logger) Warn(msg string) {
	log.core.Warn(msg)
}

func (log *Logger) Warnf(template string, args ...interface{}) {
	log.core.Sugar().Warnf(template, args...)
}

// Error log
func (log *Logger) Error(msg string) {
	log.core.Error(msg)
}

// DPanic log
func (log *Logger) DPanic(msg string) {
	log.core.DPanic(msg)
}

// Panic log
func (log *Logger) Panic(msg string) {
	log.core.Panic(msg)
}

// Fatal log
func (log *Logger) Fatal(msg string) {
	log.core.Fatal(msg)
}

// Sync flush buffered logs
func (log *Logger) Sync() error {
	return log.core.Sync()
}

// With zap fields
func With(fileds ...Field) *Logger {
	return gLogger.With(fileds...)
}

// Print log
func Print(msg string) {
	gLogger.Info(msg)
}

// Printf log
func Printf(template string, args ...interface{}) {
	gLogger.Infof(template, args...)
}

// Println log
func Println(msg string) {
	gLogger.Info(msg)
}

// Fatal log
func Fatal(msg string) {
	gLogger.Fatal(msg)
}

// Fatalf log
func Fatalf(template string, args ...interface{}) {
	gLogger.core.Sugar().Fatalf(template, args...)
}

// Fatalw log
func Fatalw(msg string, keysAndValues ...interface{}) {
	gLogger.core.Sugar().Fatalw(msg, keysAndValues...)
}

// Fatalln log
func Fatalln(args ...interface{}) {
	gLogger.core.Sugar().Fatal(args...)
}

// Panic log
func Panic(msg string) {
	gLogger.Panic(msg)
}

// Panicf log
func Panicf(template string, args ...interface{}) {
	gLogger.core.Sugar().Panicf(template, args...)
}

// Panicw log
func Panicw(msg string, keysAndValues ...interface{}) {
	gLogger.core.Sugar().Panicw(msg, keysAndValues...)
}

// Debug log
func Debug(msg string) {
	gLogger.Debug(msg)
}

// Debugf log
func Debugf(template string, args ...interface{}) {
	gLogger.core.Sugar().Debugf(template, args...)
}

// Debugw log
func Debugw(msg string, keysAndValues ...interface{}) {
	gLogger.core.Sugar().Debugw(msg, keysAndValues...)
}

// Info log
func Info(msg string) {
	gLogger.Info(msg)
}

// Infof log
func Infof(template string, args ...interface{}) {
	gLogger.Infof(template, args...)
}

// Infow log
func Infow(msg string, keysAndValues ...interface{}) {
	gLogger.core.Sugar().Infow(msg, keysAndValues...)
}

// Warn log
func Warn(msg string) {
	gLogger.Warn(msg)
}

// Warnf log
func Warnf(template string, args ...interface{}) {
	gLogger.Warnf(template, args...)
}

// Warnw log
func Warnw(msg string, keysAndValues ...interface{}) {
	gLogger.core.Sugar().Warnw(msg, keysAndValues...)
}

// Error log
func Error(msg string) {
	gLogger.Error(msg)
}

// Errorf log
func Errorf(template string, args ...interface{}) {
	gLogger.core.Sugar().Errorf(template, args...)
}

// Errorw log
func Errorw(msg string, keysAndValues ...interface{}) {
	gLogger.core.Sugar().Errorw(msg, keysAndValues...)
}
