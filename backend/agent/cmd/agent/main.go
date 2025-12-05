package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/netly/agent/config"
	"github.com/netly/agent/internal/communicator"
	"github.com/netly/agent/internal/executor"
	"github.com/netly/agent/internal/stats"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	Version = "0.1.0"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Initialize logger
	logger := initLogger(cfg.LogPath)
	defer logger.Sync()

	logger.Info("starting netly agent",
		zap.String("version", Version),
		zap.String("backend", cfg.BackendURL),
	)

	// Initialize components
	collector := stats.NewCollector()
    client := communicator.NewClient(communicator.ClientConfig{
        BackendURL: cfg.BackendURL,
        NodeToken:  cfg.NodeToken,
        Version:    Version,
        Logger:     logger,
    })
	processor := executor.NewProcessor(logger)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start heartbeat loop
	ticker := time.NewTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()

	logger.Info("heartbeat loop started",
		zap.Duration("interval", cfg.HeartbeatInterval),
	)

	// Initial heartbeat
	sendHeartbeat(logger, collector, client, processor)

	for {
		select {
		case <-ticker.C:
			sendHeartbeat(logger, collector, client, processor)

		case sig := <-quit:
			logger.Info("received shutdown signal", zap.String("signal", sig.String()))
			logger.Info("agent stopped gracefully")
			return
		}
	}
}

func sendHeartbeat(logger *zap.Logger, collector *stats.Collector, client *communicator.Client, processor *executor.Processor) {
	systemStats, err := collector.Collect()
	if err != nil {
		logger.Warn("failed to collect stats", zap.Error(err))
		return
	}

	logger.Debug("collected stats",
		zap.Float64("cpu", systemStats.CPUUsage),
		zap.Float64("ram", systemStats.RAMUsage),
		zap.Uint64("uptime", systemStats.Uptime),
	)

	resp, err := client.SendHeartbeat(systemStats)
	if err != nil {
		// Don't crash - just log and retry next tick
		logger.Warn("heartbeat failed", zap.Error(err))
		return
	}

	logger.Debug("heartbeat sent successfully")

	if len(resp.Commands) > 0 {
		logger.Info("received commands from backend",
			zap.Int("count", len(resp.Commands)),
		)

		// Process each command
		for _, cmd := range resp.Commands {
			// Convert communicator.Command to executor.Command
			execCmd := executor.Command{
				ID:        cmd.ID,
				Type:      cmd.Type,
				Payload:   cmd.Payload,
				Priority:  cmd.Priority,
				CreatedAt: cmd.CreatedAt,
			}

			result := processor.Execute(execCmd)

			// Report result back to backend (best effort)
			if err := client.ReportCommandResult(result.CommandID, result.Success, result.Output); err != nil {
				logger.Warn("failed to report command result",
					zap.Uint("command_id", result.CommandID),
					zap.Error(err),
				)
			}
		}
	}
}

func initLogger(logPath string) *zap.Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Console output for development
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)

	cores := []zapcore.Core{consoleCore}

	// File output if path is specified and writable
	if logPath != "" {
		if file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
			fileCore := zapcore.NewCore(
				jsonEncoder,
				zapcore.AddSync(file),
				zapcore.InfoLevel,
			)
			cores = append(cores, fileCore)
		}
	}

	core := zapcore.NewTee(cores...)
	return zap.New(core, zap.AddCaller())
}
