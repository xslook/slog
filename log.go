package slog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.Logger
var once sync.Once

const (
	minFlushTick = 100 * time.Millisecond
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
		filename, err := normalizeFilepath(opts.Dir, opts.Filename)
		if err != nil {
			return err
		}
		outWriter = zapcore.AddSync(&lumberjack.Logger{
			Filename:   filename,
			LocalTime:  true,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
		})
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
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), outWriter, opts.Level)
	samplerCore := zapcore.NewSampler(core, time.Second, 100, 100)
	logger = zap.New(samplerCore, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.DPanicLevel))
	zap.ReplaceGlobals(logger)

	once.Do(func() {
		go func() {
			flushTick := opts.FlushTick
			if flushTick < minFlushTick {
				flushTick = minFlushTick
			}
			ticker := time.NewTicker(flushTick)
			for range ticker.C {
				if logger != nil {
					logger.Sync()
				}
			}
		}()
	})
	return nil
}

type Level = zapcore.Level

const (
	DebugLevel Level = zapcore.DebugLevel
	InfoLevel  Level = zapcore.InfoLevel
	WarnLevel  Level = zapcore.WarnLevel
	ErrorLevel Level = zapcore.ErrorLevel
	FatalLevel Level = zapcore.FatalLevel
)

// Options for logger
type Options struct {
	Dir       string
	Filename  string
	Level     Level
	Rotate    bool // rotate log file or not
	LocalTime bool
	Stdout    bool

	FlushTick time.Duration

	MaxSize    int // The max size of single log file, default 200
	MaxBackups int // The max backup number of files
	MaxAge     int // The max keep days of log files
}

// Init a logger
func Init(opts *Options) *zap.Logger {

	initLogger(opts)

	return nil
}

func defaultOptions() *Options {
	dir := os.Getenv("LOG_DIR")
	if dir == "" {
		dir = "."
	}
	filename := os.Getenv("LOG_FILE")

	var logLevel Level
	level := os.Getenv("LOG_LEVEL")
	switch strings.ToLower(level) {
	case "debug":
		logLevel = DebugLevel
	case "info":
		logLevel = InfoLevel
	case "warn":
		logLevel = WarnLevel
	case "error":
		logLevel = ErrorLevel
	case "fatal":
		logLevel = FatalLevel
	default:
		logLevel = InfoLevel
	}
	stdout := filename == ""
	opts := &Options{
		Dir:        dir,
		Filename:   filename,
		Level:      logLevel,
		LocalTime:  true,
		Stdout:     stdout,
		MaxSize:    200,
		MaxAge:     2,
		MaxBackups: 2,
	}
	return opts
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

// Info log
func Info(args ...interface{}) {
	getLogger().Sugar().Info(args...)
}

// Infof log
func Infof(template string, args ...interface{}) {
	getLogger().Sugar().Infof(template, args...)
}

// Warn log
func Warn(args ...interface{}) {
	getLogger().Sugar().Warn(args...)
}

// Warnf log
func Warnf(template string, args ...interface{}) {
	getLogger().Sugar().Warnf(template, args...)
}

// Error log
func Error(args ...interface{}) {
	getLogger().Sugar().Error(args...)
}

// Errorf log
func Errorf(template string, args ...interface{}) {
	getLogger().Sugar().Errorf(template, args...)
}
