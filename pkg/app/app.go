package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/Essen-Labs/bloom-be/pkg/config"
	"github.com/Essen-Labs/bloom-be/pkg/handler"
	"github.com/Essen-Labs/bloom-be/pkg/middleware"
	"github.com/Essen-Labs/bloom-be/pkg/validator"
	"github.com/Essen-Labs/bloom-be/translation"
	"github.com/dwarvesf/gerr"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	_ "github.com/Essen-Labs/bloom-be/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// App api app instance
type App struct {
	cfg config.Config
	l   gerr.Log
	th  translation.Helper
	db  *sql.DB
}

// LoadApp load config and init app
func LoadApp(db *sql.DB) *App {
	cls := config.DefaultConfigLoaders()
	cfg := config.LoadConfig(cls)
	l := gerr.NewSimpleLog()
	th := translation.NewTranslatorHelper()

	return &App{
		cfg: cfg,
		l:   l,
		th:  th,
		db:  db,
	}
}

// Run api app
func (a App) Run() {
	router := a.setupRouter()
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	err := a.createTables()
	if err != nil {
		log.Fatal("Error creating tables: ", err)
		return
	}
	gerr.SetCleanPathFunc(func(path string) string {
		projName := "bloom-be/"
		startIdx := strings.Index(path, projName)
		if startIdx >= 0 {
			path = path[startIdx+len(projName):]
		}
		return path
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", a.cfg.Port),
		Handler: router,
	}

	go func() {
		// service connections
		a.l.Info("listening on ", a.cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
		quit <- os.Interrupt
	}()

	select {
	case <-quit:

		a.l.Info("Shutdown Server ...")
		ctx, cancel := context.WithTimeout(context.Background(), a.cfg.GetShutdownTimeout())
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			a.l.Error("Server Shutdown:", err)
		}
		a.l.Info("Server exiting")
	}
}

func (a App) setupRouter() *gin.Engine {
	r := gin.New()
	binding.Validator = validator.NewStructValidator(a.th)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Use(middleware.NewLogDataMiddleware(a.cfg.ServiceName, a.cfg.Env))
	r.Use(cors.New(
		cors.Config{
			AllowOrigins: a.cfg.GetCORS(),
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
			AllowHeaders: []string{"Origin", "Host",
				"Content-Type", "Content-Length",
				"Accept-Encoding", "Accept-Language", "Accept",
				"X-CSRF-Token", "Authorization", "X-Requested-With", "X-Access-Token"},
			ExposeHeaders:    []string{"MeAllowMethodsntent-Length"},
			AllowCredentials: true,
		},
	))

	h := handler.NewHandler(a.cfg, a.l, a.th, a.db)

	// handlers
	r.GET("/healthz", h.Healthz)
	r.GET("/get-chat-by-id", h.GetChatById)
	r.GET("/get-chat-list", h.GetAllChat)
	r.POST("/send-chat", h.Completions)
	r.GET("/get-all-msgs-by-id", h.GetAllMsgsByID)
	r.POST("/delete-chat", h.DeleteChatById)
	r.POST("/delete-all-chat", h.DeleteAllChat)
	// r.POST("/edit-chat", h.Completions)         // TODO
	// r.GET("/get-model-list", h.Completions)     // TODO
	return r
}
