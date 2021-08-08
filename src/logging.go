package src

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger

func InitLogging(dev bool) {
	level := zap.InfoLevel
	if dev {
		level = zap.DebugLevel
	}
	logger, _ := zap.Config{
		Development:      false,
		Level:            zap.NewAtomicLevelAt(level),
		Encoding:         "console",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
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
	defer logger.Sync() // flushes buffer, if any
	Logger = logger.Sugar()
}
