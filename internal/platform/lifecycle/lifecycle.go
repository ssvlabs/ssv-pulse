package lifecycle

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const terminationDelay = time.Second

func ListenForApplicationShutDown(shutdownFunc func(), signalChannel chan os.Signal) {
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	sig := <-signalChannel
	switch sig {
	case os.Interrupt, syscall.SIGTERM:
		slog.Info("shutdown signal received")
		shutdownFunc()
		time.Sleep(terminationDelay)
	}
}

func ShutDown(shutdownFunc func(), code int) {
	shutdownFunc()
	time.Sleep(terminationDelay)
	slog.Warn("application was terminated")
	os.Exit(code)
}
