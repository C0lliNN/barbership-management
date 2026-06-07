// Command api is the entrypoint for the Barbershop Management Platform HTTP API.
package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/gcollin65/barbershop/internal/config"
	"github.com/gcollin65/barbershop/internal/database"
	apihttp "github.com/gcollin65/barbershop/internal/infra/http"
	"github.com/gcollin65/barbershop/internal/identity"
	"github.com/gcollin65/barbershop/internal/infra/repository"
	"github.com/gcollin65/barbershop/internal/logger"
)

func main() {
	fx.New(
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log.Named("fx")}
		}),
		fx.Provide(
			config.Load,
			newLogger,
			newPool,
		),
		fx.Provide(
			fx.Annotate(repository.NewShopRepository, fx.As(new(identity.ShopRepository))),
			fx.Annotate(repository.NewUserRepository, fx.As(new(identity.UserRepository))),
			fx.Annotate(repository.NewMembershipRepository, fx.As(new(identity.MembershipRepository))),
		),
		fx.Provide(
			newIdentityService,
			func(svc *identity.Service) identity.Signer { return svc },
			func(svc *identity.Service) identity.ShopManager { return svc },
			newRouter,
		),
		fx.Invoke(
			runMigrations,
			registerRoutes,
			runServer,
		),
	).Run()
}

func newLogger(lc fx.Lifecycle, cfg config.Config) (*zap.Logger, error) {
	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			_ = log.Sync()
			return nil
		},
	})
	return log, nil
}

func newPool(lc fx.Lifecycle, cfg config.Config, log *zap.Logger) (*pgxpool.Pool, error) {
	pool, err := database.New(context.Background(), &cfg, log)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			database.Close(pool)
			return nil
		},
	})
	return pool, nil
}

func newIdentityService(
	shops identity.ShopRepository,
	users identity.UserRepository,
	members identity.MembershipRepository,
	cfg config.Config,
) *identity.Service {
	return identity.NewService(shops, users, members,
		identity.WithJWTSecret(cfg.JWTSecret),
		identity.WithJWTExpiry(cfg.JWTExpiry),
	)
}

func newRouter(cfg config.Config, log *zap.Logger, pool *pgxpool.Pool) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	return apihttp.NewRouter(log, pool, cfg.CORSAllowedOrigins)
}

func runMigrations(cfg config.Config, log *zap.Logger) error {
	return database.RunMigrations(cfg.DatabaseURL, database.Migrations, log)
}

func registerRoutes(
	engine *gin.Engine,
	pool *pgxpool.Pool,
	signer identity.Signer,
	shops identity.ShopManager,
	memberships identity.MembershipRepository,
	cfg config.Config,
) {
	v1 := engine.Group("/v1", apihttp.Transaction(pool))
	apihttp.RegisterIdentityRoutes(v1, signer, cfg.JWTSecret)
	apihttp.RegisterShopRoutes(v1, shops, memberships, cfg.JWTSecret)
}

func runServer(lc fx.Lifecycle, engine *gin.Engine, cfg config.Config, log *zap.Logger) {
	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      engine,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				log.Info("server starting", zap.Int("port", cfg.Port), zap.String("env", cfg.Env))
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Error("server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("server stopping")
			return srv.Shutdown(ctx)
		},
	})
}
