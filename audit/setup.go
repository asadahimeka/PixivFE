// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package audit

import (
	"fmt"
	"os"

	"codeberg.org/pixivfe/pixivfe/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	maxRecordedResponses   int         = 128
	responseDirPermissions os.FileMode = 0o700
)

var GlobalAuditor *Auditor = &Auditor{Logger: zap.S()} // default value, to be used in tests

type Auditor struct {
	Logger        *zap.SugaredLogger // If the original Logger is needed, call the Desugar() method on this
	SaveResponses bool
	MaxRecorded   int
	responsePath  string
}

func Setup(cfg *config.ServerConfig) {
	zapConfig := createLoggerConfig(cfg)

	logger, err := zapConfig.Build()
	if err != nil {
		panic(fmt.Errorf("failed to initialize zap logger: %w", err))
	}

	// TODO: SaveResponses and MaxRecorded should be exposed via explicit ServerConfig options
	auditor := &Auditor{
		Logger:        logger.Sugar(),
		SaveResponses: cfg.Development.InDevelopment,
		MaxRecorded:   maxRecordedResponses,
		responsePath:  cfg.Development.ResponseSaveLocation,
	}

	if auditor.SaveResponses {
		if err := os.MkdirAll(auditor.responsePath, responseDirPermissions); err != nil {
			auditor.Logger.Fatalw("Failed to create response directory",
				"error", err,
				"path", auditor.responsePath,
			)
		}
	}

	GlobalAuditor = auditor
}

func createLoggerConfig(cfg *config.ServerConfig) zap.Config {
	zapConfig := zap.NewProductionConfig()

	zapConfig.DisableStacktrace = true // the stacktraces are useless. every log message shows the file logger.Error is called.

	// Set log format
	if cfg.Log.Format == "json" {
		zapConfig.Encoding = "json"
	} else {
		zapConfig.Encoding = "console"
	}

	// Set log level
	if level, err := zapcore.ParseLevel(cfg.Log.Level); err == nil {
		zapConfig.Level = zap.NewAtomicLevelAt(level)
	}

	// Set output paths
	if len(cfg.Log.Outputs) > 0 {
		zapConfig.OutputPaths = cfg.Log.Outputs
	}

	// Configure human-readable timestamps
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return zapConfig
}
