package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/3eLLenKa/test-avito/internal/app"
	"github.com/3eLLenKa/test-avito/internal/config"
)

func main() {
	cfg := config.MustLoad()

	application := app.NewApp(cfg)

	go func() {
		application.Server.Run()
	}()

	slog.Info("application started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	application.Server.Stop(ctx)
}
