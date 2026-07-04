package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/douyin-clone/backend/internal/config"
	"github.com/CodingFervor/douyin-clone/backend/internal/db"
	"github.com/CodingFervor/douyin-clone/backend/internal/handler"
	"github.com/CodingFervor/douyin-clone/backend/internal/repository"
	"github.com/CodingFervor/douyin-clone/backend/internal/seed"
	"github.com/CodingFervor/douyin-clone/backend/internal/server"
)

func main() {
	cfg := config.Load()
	gin.SetMode(gin.ReleaseMode)

	if err := db.Init(cfg.DBPath); err != nil {
		os.Stderr.WriteString("init db: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer db.Close()

	seed.Run(db.DB)

	h := handler.New(cfg.JWTSecret,
		repository.NewUserRepo(db.DB),
		repository.NewVideoRepo(db.DB),
		repository.NewLikeRepo(db.DB),
		repository.NewCommentRepo(db.DB),
		repository.NewFollowRepo(db.DB),
		repository.NewFavoriteRepo(db.DB),
		repository.NewRecommendRepo(db.DB),
		repository.NewNotifyRepo(db.DB),
	)
	// Seed live rooms + products.
	repository.NewLiveRepo(db.DB).SeedLive()
	repository.NewLiveRepo(db.DB).SeedSchedules()

	// Seed the gift catalog + rebuild the hashtag usage counts.
	hashtagRepo := repository.NewHashtagRepo(db.DB)
	hashtagRepo.Rebuild()
	giftRepo := repository.NewGiftRepo(db.DB)
	giftRepo.SeedGifts()
	h.SetLive(repository.NewLiveRepo(db.DB), repository.NewDanmakuRepo(db.DB), repository.NewSearchLogRepo(db.DB), hashtagRepo, giftRepo)
	// Attach the private-message repo.
	h.SetMessages(repository.NewMessageRepo(db.DB))

	// Ensure the uploads directory exists for the static file server.
	_ = os.MkdirAll("data/uploads", 0o755)
	_ = os.MkdirAll("data/images", 0o755)

	r := server.New(h, cfg.AllowedOrigins)
	addr := ":" + strconv.Itoa(cfg.Port)
	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		os.Stderr.WriteString("douyin-clone server listening on " + addr + "\n")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			os.Stderr.WriteString("server failed: " + err.Error() + "\n")
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		os.Stderr.WriteString("server shutdown error: " + err.Error() + "\n")
	}
}
