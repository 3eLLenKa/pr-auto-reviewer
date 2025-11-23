package app

import (
	"fmt"

	"github.com/3eLLenKa/test-avito/internal/config"
	api "github.com/3eLLenKa/test-avito/internal/delivery/http/gen"
	"github.com/3eLLenKa/test-avito/internal/delivery/http/handlers"
	"github.com/3eLLenKa/test-avito/internal/delivery/http/server"
	"github.com/3eLLenKa/test-avito/internal/repository"
	"github.com/3eLLenKa/test-avito/internal/repository/postgres"
	"github.com/3eLLenKa/test-avito/internal/service"
	"github.com/gin-gonic/gin"
)

type App struct {
	Server *server.Server
}

func NewApp(cfg *config.Config) *App {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.DBName,
		cfg.Database.Password,
		cfg.Database.SSLMode,
	)

	pg, err := postgres.New(dsn)
	if err != nil {
		panic(err)
	}

	repo := repository.New(pg.Db)
	svc := service.New(repo.PullRequest, repo.Team, repo.User)

	handler := api.NewStrictHandler(
		handlers.NewHandlers(svc),
		nil,
	)

	router := gin.New()
	router.Use(gin.Recovery())
	api.RegisterHandlers(router, handler)

	addr := ":" + cfg.App.Port

	httpServer := server.New(addr, router)

	return &App{
		Server: httpServer,
	}
}
