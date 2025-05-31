// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/assets"
	"codeberg.org/pixivfe/pixivfe/server/middleware"
	"codeberg.org/pixivfe/pixivfe/server/middleware/limiter"
	"codeberg.org/pixivfe/pixivfe/server/router"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

const (
	// Values for http.Server timeouts.
	// ref: gosec: G112
	readHeaderTimeout time.Duration = 15 * time.Second
	readTimeout       time.Duration = 15 * time.Second
	writeTimeout      time.Duration = 10 * time.Second
	idleTimeout       time.Duration = 30 * time.Second

	serverShutdownDeadline time.Duration = 5 * time.Second
)

// embeddedContent holds our static web server content.
//
//go:embed all:assets all:i18n
var embeddedContent embed.FS

func init() {
	// Assign the embedded filesystem to the exported assets.FS variable.
	assets.FS = embeddedContent
}

//nolint:funlen
func main() {
	if err := config.GlobalConfig.LoadConfig(); err != nil {
		panic(fmt.Errorf("failed to load configuration: %w", err))
	}

	audit.Setup(&config.GlobalConfig)
	audit.GlobalAuditor.Logger.Info("Auditor initialized")

	if err := i18n.Setup(); err != nil {
		audit.GlobalAuditor.Logger.Errorf("Failed to initialize i18n engine: %v", err)
	}

	audit.GlobalAuditor.Logger.Info("i18n engine initialized")

	template.Setup(config.GlobalConfig.Development.InDevelopment)

	if err := template.LoadIcons("assets/icons"); err != nil {
		audit.GlobalAuditor.Logger.Errorf("Failed to load icons: %v", err)
	}

	// Initialize cache
	requests.Setup()
	audit.GlobalAuditor.Logger.Info("API response cache initialized.")

	audit.GlobalAuditor.Logger.Info("Starting server...")

	router := router.DefineRoutes()
	// the first middleware is the most outer / first executed one
	router.Use(middleware.WithRequestContext)  // needed for everything else
	router.Use(middleware.SetLocaleFromCookie) // needed for i18n.*()
	router.Use(middleware.SetResponseHeaders)  // all pages need this
	router.Use(middleware.HandleError)         // if the inner handler fails, this shows the error page instead

	// Limiter setup
	if config.GlobalConfig.Limiter.Enabled {
		limiter.Setup()
		router.Use(limiter.Evaluate)
	}

	// watch and compile tailwind css when in development mode
	if config.GlobalConfig.Development.InDevelopment {
		go watchTailwindCSS()
	}

	// Create http.Server instance
	server := &http.Server{
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Listen
	listener := chooseListener()

	// Start main server
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			audit.GlobalAuditor.Logger.Fatalf("Starting http.Server failed: %v", err)
		}
	}()

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	audit.GlobalAuditor.Logger.Info("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), serverShutdownDeadline)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		audit.GlobalAuditor.Logger.Fatalf("Server forced to shutdown: %v", err)
	}

	audit.GlobalAuditor.Logger.Info("Server exiting")
}

func chooseListener() net.Listener {
	var listener net.Listener

	// Check if we should use a Unix domain socket
	if config.GlobalConfig.Basic.UnixSocket != "" {
		unixAddr := config.GlobalConfig.Basic.UnixSocket

		unixListener, err := net.Listen("unix", unixAddr)
		if err != nil {
			audit.GlobalAuditor.Logger.Panicf("Failed to start Unix socket listener on %v: %v", unixAddr, err)
		}

		// Assign the listener and log where we are listening
		listener = unixListener

		audit.GlobalAuditor.Logger.Infof("Listening on Unix domain socket: %v", unixAddr)
	} else {
		// Otherwise, fall back to TCP listener
		addr := net.JoinHostPort(config.GlobalConfig.Basic.Host, config.GlobalConfig.Basic.Port)

		tcpListener, err := net.Listen("tcp", addr)
		if err != nil {
			audit.GlobalAuditor.Logger.Panicf("Failed to start TCP listener on %v: %v", addr, err)
		}

		// Assign the TCP listener
		listener = tcpListener
		addr = tcpListener.Addr().String()

		// Extract the host and port for logging
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			audit.GlobalAuditor.Logger.Panicf("Failed to parse listener address %q: %v", addr, err)
		}

		// Log the address and convenient URL for local development
		audit.GlobalAuditor.Logger.Infof("Listening on %v. Accessible at: http://pixivfe.localhost:%v/", addr, port)
	}

	return listener
}

func watchTailwindCSS() {
	cmd := exec.Command(
		"tailwindcss",
		"-i", "assets/css/tailwind-style_source.css",
		"-o", "assets/css/tailwind-style.css",
		"--watch",
		"--minify")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		audit.GlobalAuditor.Logger.Errorf("Error running tailwindcss command: %v", err)

		return
	}
}
