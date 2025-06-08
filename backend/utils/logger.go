package utils

import (
	"os"
	"time"

	"github.com/fynelabs/selfupdate"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger      *zap.SugaredLogger
	WailsLogger *ZapAdapter
	Logfile     *lumberjack.Logger
)

// ZapAdapter wraps a zap.SugaredLogger to conform to the logger.Logger interface.
type ZapAdapter struct {
	sugar *zap.SugaredLogger
}

func (z *ZapAdapter) Print(message string)   { z.sugar.Info(message) }
func (z *ZapAdapter) Trace(message string)   { z.sugar.Debug(message) }
func (z *ZapAdapter) Debug(message string)   { z.sugar.Debug(message) }
func (z *ZapAdapter) Info(message string)    { z.sugar.Info(message) }
func (z *ZapAdapter) Warning(message string) { z.sugar.Warn(message) }
func (z *ZapAdapter) Error(message string)   { z.sugar.Error(message) }
func (z *ZapAdapter) Fatal(message string)   { z.sugar.Fatal(message) }

func InitLogger() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	if err := os.MkdirAll("./logs", 0755); err != nil {
		panic(err)
	}
	Logfile = &lumberjack.Logger{
		Filename:   "./logs/dacapo.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		LocalTime:  true,
	}

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
		zapcore.NewCore(fileEncoder, zapcore.AddSync(Logfile), zapcore.DebugLevel),
	)

	slogger := zap.New(core)
	Logger = slogger.Sugar()
	WailsLogger = &ZapAdapter{sugar: Logger}
	selfupdate.LogError = Logger.Errorf
	selfupdate.LogInfo = Logger.Infof
	selfupdate.LogDebug = Logger.Debugf

	Logger.Info("Logger initialized")
}
