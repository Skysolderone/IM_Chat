package model

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger      *log.Logger
	logFile     *os.File
	logWriter   io.Writer
	currentDate string
)

// InitLogger 初始化日志，根据网关ID生成对应的日志文件
func InitLogger(gatewayID string) error {
	// 日志目录
	logDir := "/app/logs"
	if logDirEnv := os.Getenv("LOG_DIR"); logDirEnv != "" {
		logDir = logDirEnv
	}
	
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}
	
	// 生成日志文件名：gateway1-2024-01-01.log
	currentDate = time.Now().Format("2006-01-02")
	logFileName := fmt.Sprintf("%s-%s.log", gatewayID, currentDate)
	logFilePath := filepath.Join(logDir, logFileName)
	
	// 使用 lumberjack 实现日志轮转
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    100,   // 每个文件最大100MB
		MaxBackups: 10,   // 保留10个备份文件
		MaxAge:     30,   // 保留30天
		Compress:   true, // 压缩旧文件
		LocalTime:  true,
	}
	
	// 同时输出到文件和控制台
	logWriter = io.MultiWriter(os.Stdout, lumberjackLogger)
	
	// 创建 logger
	Logger = log.New(logWriter, "", log.LstdFlags|log.Lshortfile)
	
	Logger.Printf("日志系统初始化成功，日志文件: %s", logFilePath)
	
	// 启动定时任务检查日期变化，切换日志文件
	go checkDateChange(gatewayID, logDir)
	
	return nil
}

// checkDateChange 检查日期变化，自动切换日志文件
func checkDateChange(gatewayID, logDir string) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		today := time.Now().Format("2006-01-02")
		if today != currentDate {
			// 日期变化，重新初始化日志
			currentDate = today
			logFileName := fmt.Sprintf("%s-%s.log", gatewayID, currentDate)
			logFilePath := filepath.Join(logDir, logFileName)
			
			lumberjackLogger := &lumberjack.Logger{
				Filename:   logFilePath,
				MaxSize:    100,
				MaxBackups: 10,
				MaxAge:     30,
				Compress:   true,
				LocalTime:  true,
			}
			
			logWriter = io.MultiWriter(os.Stdout, lumberjackLogger)
			Logger = log.New(logWriter, "", log.LstdFlags|log.Lshortfile)
			Logger.Printf("日志文件已切换到: %s", logFilePath)
		}
	}
}

// CloseLogger 关闭日志文件
func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}
