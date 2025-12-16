package utils

import (
	"log"
	"os"
	"path"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	hertzzap "github.com/hertz-contrib/logger/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func SetupLogger(level hlog.Level) *hertzzap.Logger {
	logFilePath := "./logs/"
	if err := os.MkdirAll(logFilePath, 0o777); err != nil {
		log.Println(err.Error())
		return nil
	}

	// Set filename to date
	logFileName := time.Now().Format("2006-01-02") + ".log"
	fileName := path.Join(logFilePath, logFileName)
	if _, err := os.Stat(fileName); err != nil {
		if _, err := os.Create(fileName); err != nil {
			log.Println(err.Error())
			return nil
		}
	}

	// Provides compression and deletion
	lumberjackLogger := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    20,    // A file can be up to 20M.
		MaxBackups: 5,     // Save up to 5 files at the same time.
		MaxAge:     10,    // A file can exist for a maximum of 10 days.
		Compress:   false, // Compress with gzip.
	}

	// 终端输出：console encoder + 彩色 level
	consoleEncCfg := zap.NewDevelopmentEncoderConfig()
	consoleEncCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEnc := zapcore.NewConsoleEncoder(consoleEncCfg)

	// 文件输出：json encoder（不带颜色，便于检索/采集）
	fileEncCfg := zap.NewProductionEncoderConfig()
	fileEncCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEnc := zapcore.NewJSONEncoder(fileEncCfg)

	// 两个 core 共用同一个 level 控制
	var zapLvl zapcore.Level
	switch level {
	case hlog.LevelTrace, hlog.LevelDebug:
		zapLvl = zap.DebugLevel
	case hlog.LevelInfo:
		zapLvl = zap.InfoLevel
	case hlog.LevelWarn, hlog.LevelNotice:
		zapLvl = zap.WarnLevel
	case hlog.LevelError:
		zapLvl = zap.ErrorLevel
	case hlog.LevelFatal:
		zapLvl = zap.FatalLevel
	default:
		zapLvl = zap.WarnLevel
	}
	atomicLevel := zap.NewAtomicLevelAt(zapLvl)

	// For zap detailed settings, please refer to https://github.com/hertz-contrib/logger/tree/main/zap and https://github.com/uber-go/zap
	// hlog will warp a layer of zap, so you need to calculate the depth of the caller file separately.
	logger := hertzzap.NewLogger(
		hertzzap.WithCores(
			hertzzap.CoreConfig{
				Enc: fileEnc,
				Ws:  zapcore.AddSync(lumberjackLogger),
				Lvl: atomicLevel,
			},
			hertzzap.CoreConfig{
				Enc: consoleEnc,
				Ws:  zapcore.AddSync(os.Stdout),
				Lvl: atomicLevel,
			},
		),
		hertzzap.WithZapOptions(zap.AddCaller(), zap.AddCallerSkip(3)),
	)
	return logger
}
