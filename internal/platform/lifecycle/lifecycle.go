package lifecycle

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const terminationDelay = time.Second

func ListenForApplicationShutDown(ctx context.Context, shutdownFunc func(), signalChannel chan os.Signal) {
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-signalChannel:
		switch sig {
		case os.Interrupt, syscall.SIGTERM:
			slog.Warn("shutdown signal received")
			shutdownFunc()
			time.Sleep(terminationDelay)
		}
	case <-ctx.Done():
		slog.Warn("context deadline exceeded or canceled")
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
