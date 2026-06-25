package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"p-box/backend/config"
	"p-box/backend/server"
)

var (
	Version   = "2.0.3"
	BuildTime = "unknown"
)

func main() {
	port := flag.Int("port", 8383, "API service port")
	configPath := flag.String("config", "config.yaml", "config file path")
	debug := flag.Bool("debug", false, "enable debug mode")
	showVersion := flag.Bool("version", false, "show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("P-BOX v%s (Build: %s)\n", Version, BuildTime)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	if *port != 8383 {
		cfg.Server.Port = *port
	}
	if *debug {
		cfg.Log.Level = "debug"
	}

	server.Version = Version
	server.BuildTime = BuildTime

	srv := server.New(cfg)
	go func() {
		if err := srv.Start(); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			fmt.Printf("server startup failed: %v\n", err)
			os.Exit(1)
		}
	}()

	fmt.Printf("P-BOX v%s started\n", Version)
	fmt.Printf("API address: http://localhost:%d\n", cfg.Server.Port)
	fmt.Printf("Web panel: http://localhost:%d\n", cfg.Server.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down service...")
	srv.Shutdown()
	fmt.Println("Service stopped")
}
