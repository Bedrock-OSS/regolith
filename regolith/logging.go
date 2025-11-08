package regolith

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger
var LoggerLevel zap.AtomicLevel

type colorWriter struct {
	io.Writer
}

func (cw colorWriter) Close() error {
	return nil
}
func (cw colorWriter) Sync() error {
	return nil
}

func InitLogging(dev bool) {
	if Logger != nil {
		return
	}
	_, b := os.LookupEnv("FORCE_COLOR")
	if b {
		color.NoColor = false
	}
	err := zap.RegisterSink("color", func(url *url.URL) (zap.Sink, error) {
		if url.Host == "stderr" {
			return colorWriter{color.Output}, nil
		}
		return colorWriter{color.Output}, nil
	})
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	LoggerLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	if dev {
		LoggerLevel.SetLevel(zap.DebugLevel)
	}
	logger, _ := zap.Config{
		Development:       dev,
		Level:             LoggerLevel,
		Encoding:          "console",
		OutputPaths:       []string{"color:stdout"},
		ErrorOutputPaths:  []string{"color:stderr"},
		DisableStacktrace: true,
		DisableCaller:     true,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:       "T",
			LevelKey:      "L",
			NameKey:       "N",
			CallerKey:     "C",
			FunctionKey:   zapcore.OmitKey,
			MessageKey:    "M",
			StacktraceKey: "S",
			LineEnding:    zapcore.DefaultLineEnding,
			// Color level and put it into brackets
			EncodeLevel: func(level zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {
				var result string
				switch level {
				case zap.InfoLevel:
					result = fmt.Sprintf("[%s]", color.CyanString(level.CapitalString()))
				case zap.DebugLevel:
					result = fmt.Sprintf("[%s]", color.BlueString(level.CapitalString()))
				case zap.WarnLevel:
					result = fmt.Sprintf("[%s]", color.YellowString(level.CapitalString()))
				case zap.ErrorLevel:
					result = fmt.Sprintf("[%s]", color.RedString(level.CapitalString()))
				case zap.FatalLevel:
					result = fmt.Sprintf("[%s]", color.RedString(level.CapitalString()))
				case zap.PanicLevel:
				case zap.DPanicLevel:
					result = fmt.Sprintf("[%s]", color.New(color.FgRed, color.BgWhite).Sprint(level.CapitalString()))
				}
				encoder.AppendString(result)
			},
			// Hide time
			EncodeTime: func(time time.Time, encoder zapcore.PrimitiveArrayEncoder) {

			},
			EncodeDuration: zapcore.StringDurationEncoder,
			// Hide caller
			EncodeCaller: func(caller zapcore.EntryCaller, encoder zapcore.PrimitiveArrayEncoder) {

			},
		},
	}.Build()
	Logger = logger.Sugar()
}

// ShutdownLogging flushes any buffered log entries. It should be called
// before the program exits.
func ShutdownLogging() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}
