package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"qtp/internal/attempt"
	"qtp/internal/config"
	"qtp/internal/database"
	"qtp/internal/httpx"
	"qtp/internal/mailer"
	"qtp/internal/question"
	"qtp/internal/questionbank"
	"qtp/internal/questionimport"
	"qtp/internal/quiz"
	"qtp/internal/session"
	"qtp/internal/user"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	databaseContext, cancelDatabase := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	pool, err := database.Open(databaseContext, cfg.Database)
	cancelDatabase()
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	//QUESTION
	questionRepository := question.NewQuestionRepository(pool)
	questionService := question.NewQuestionService(questionRepository)
	questionHandler := question.NewQuestionHandler(questionService)

	//QUESTION BANK
	questionBankRepository := questionbank.NewQuestionBankRepository(pool)
	questionBankService := questionbank.NewQuestionBankService(questionBankRepository)
	questionBankHandler := questionbank.NewQuestionBankHandler(questionBankService)

	// USER
	userRepository := user.NewUserRepository(pool)
	sessionRepository := session.NewSessionRepository(pool)

	emailSender := mailer.NewSMTP(cfg.Mail)
	userService := user.NewUserService(userRepository, sessionRepository, emailSender, cfg.Security.PublicOrigin)
	userHandler := user.NewUserHandler(userService, cfg.Security.CookieSecure)
	if cfg.Google.ClientID != "" {
		userHandler.ConfigureGoogleLogin(cfg.Google.ClientID, cfg.Google.ClientSecret, cfg.Google.RedirectURL)
	}

	// File uploading
	questionImportService := questionimport.NewService(questionBankService, questionRepository)
	questionImportHandler := questionimport.NewHandler(questionImportService)

	//QUIZ
	quizRepository := quiz.NewQuizRepository(pool)
	quizService := quiz.NewQuizService(quizRepository, questionService)
	quizHandler := quiz.NewQuizHandler(quizService)

	// ATTEMPT
	attemptRepository := attempt.NewAttemptRepository(pool)
	attemptService := attempt.NewAttemptService(attemptRepository)
	attemptHandler := attempt.NewAttemptHandler(attemptService)

	slog.Info(
		"database connected",
		"host",
		cfg.Database.Host,
		"port",
		cfg.Database.Port,
		"database",
		cfg.Database.Name,
	)

	router := http.NewServeMux()
	loginLimiter := httpx.NewRateLimiter(10, time.Minute, 10000)
	registerLimiter := httpx.NewRateLimiter(5, time.Hour, 10000)
	resendLimiter := httpx.NewRateLimiter(10, time.Hour, 10000)
	verifyLimiter := httpx.NewRateLimiter(60, time.Hour, 10000)
	googleLimiter := httpx.NewRateLimiter(30, time.Minute, 10000)
	importLimiter := httpx.NewRateLimiter(10, time.Minute, 10000)
	importKey := func(r *http.Request) string {
		currentUser, _ := user.UserFromContext(r.Context())
		return currentUser.ID
	}

	//Question handler
	router.Handle("DELETE /v1/question-banks/{bankID}/questions/{questionID}", userHandler.RequireAuth(http.HandlerFunc(questionHandler.Delete)))
	router.Handle("PUT /v1/question-banks/{bankID}/questions/{questionID}", userHandler.RequireAuth(http.HandlerFunc(questionHandler.Update)))
	router.Handle("GET /v1/question-banks/{bankID}/questions", userHandler.RequireAuth(http.HandlerFunc(questionHandler.ListByBankID)))
	router.Handle("GET /v1/question-banks/{bankID}/questions/{questionID}", userHandler.RequireAuth(http.HandlerFunc(questionHandler.FindByID)))
	router.Handle("POST /v1/question-banks/{bankID}/questions", userHandler.RequireAuth(http.HandlerFunc(questionHandler.Create)))

	//Question Bank handler
	router.Handle("GET /v1/question-banks", userHandler.RequireAuth(http.HandlerFunc(questionBankHandler.List)))
	router.Handle("GET /v1/question-banks/{bankID}", userHandler.RequireAuth(http.HandlerFunc(questionBankHandler.Find)))
	router.Handle("POST /v1/question-banks", userHandler.RequireAuth(http.HandlerFunc(questionBankHandler.Create)))
	router.Handle("PATCH /v1/question-banks/{bankID}", userHandler.RequireAuth(http.HandlerFunc(questionBankHandler.Patch)))
	router.Handle("DELETE /v1/question-banks/{bankID}", userHandler.RequireAuth(http.HandlerFunc(questionBankHandler.Delete)))

	//AUTH Routes
	router.Handle("POST /v1/auth/logout", userHandler.RequireAuth(http.HandlerFunc(userHandler.Logout)))
	router.Handle("GET /v1/auth/csrf", userHandler.RequireAuth(http.HandlerFunc(userHandler.CSRF)))
	router.Handle("GET /v1/auth/me", userHandler.RequireAuth(http.HandlerFunc(userHandler.Me)))
	router.Handle("POST /v1/auth/login", loginLimiter.Middleware(httpx.RemoteIP, http.HandlerFunc(userHandler.Login)))
	router.Handle("POST /v1/auth/register", registerLimiter.Middleware(httpx.RemoteIP, http.HandlerFunc(userHandler.Register)))
	router.Handle("GET /v1/auth/google/start", googleLimiter.Middleware(httpx.RemoteIP, http.HandlerFunc(userHandler.GoogleStart)))
	router.Handle("GET "+config.GoogleCallbackPath, googleLimiter.Middleware(httpx.RemoteIP, http.HandlerFunc(userHandler.GoogleCallback)))
	router.Handle("POST /v1/auth/verify-email", verifyLimiter.Middleware(httpx.RemoteIP, http.HandlerFunc(userHandler.VerifyEmail)))
	router.Handle("POST /v1/auth/resend-verification", resendLimiter.Middleware(httpx.RemoteIP, http.HandlerFunc(userHandler.ResendVerification)))
	router.HandleFunc("GET /v1/auth/verify-email", userHandler.VerificationPage)

	router.HandleFunc("GET /health", healthHandler)
	router.HandleFunc("GET /ready", readyHandler(pool))

	//File Uploading
	router.Handle("POST /v1/question-banks/{bankID}/imports/preview", userHandler.RequireAuth(importLimiter.Middleware(importKey, http.HandlerFunc(questionImportHandler.Preview))))
	router.Handle("POST /v1/question-banks/{bankID}/imports", userHandler.RequireAuth(importLimiter.Middleware(importKey, http.HandlerFunc(questionImportHandler.Import))))

	//Quiz
	router.Handle("POST /v1/question-banks/{bankID}/quizzes", userHandler.RequireAuth(http.HandlerFunc(quizHandler.CreateQuiz)))
	router.Handle("GET /v1/quizzes/{quizID}", userHandler.RequireAuth(http.HandlerFunc(quizHandler.FindQuizByOwnID)))
	router.Handle("GET /v1/quizzes", userHandler.RequireAuth(http.HandlerFunc(quizHandler.List)))
	router.Handle("DELETE /v1/quizzes/{quizID}", userHandler.RequireAuth(http.HandlerFunc(quizHandler.Delete)))

	//Attempt
	router.Handle("POST /v1/quizzes/{quizID}/attempts", userHandler.RequireAuth(http.HandlerFunc(attemptHandler.Start)))
	router.Handle("GET /v1/attempts/{attemptID}", userHandler.RequireAuth(http.HandlerFunc(attemptHandler.Get)))
	router.Handle("PUT /v1/attempts/{attemptID}/answers/{quizQuestionID}", userHandler.RequireAuth(http.HandlerFunc(attemptHandler.SaveAnswer)))

	router.Handle("POST /v1/attempts/{attemptID}/finish", userHandler.RequireAuth(http.HandlerFunc(attemptHandler.Finish)))
	router.Handle("GET /v1/attempts/{attemptID}/result", userHandler.RequireAuth(http.HandlerFunc(attemptHandler.GetResult)))
	router.Handle("GET /v1/quizzes/{quizID}/attempts", userHandler.RequireAuth(http.HandlerFunc(attemptHandler.GetHistory)))

	server := &http.Server{
		Addr: net.JoinHostPort(
			"",
			strconv.Itoa(int(cfg.HTTP.Port)),
		),
		Handler:           httpx.Observe(httpx.TrustedOrigin(cfg.Security.PublicOrigin, httpx.JSONRouter(router))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	serverError := make(chan error, 1)

	go func() {
		slog.Info(
			"server started",
			"application",
			cfg.AppName,
			"address",
			server.Addr,
		)

		serverError <- server.ListenAndServe()
	}()

	signalContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("listen and serve: %w", err)

	case <-signalContext.Done():
		slog.Info("shutdown signal received")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownContext); err != nil {
		// Force connections closed on grace expiry before waiting for the pool.
		_ = server.Close()
		return fmt.Errorf("shutdown server: %w", err)
	}

	err = <-serverError
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server stopped: %w", err)
	}

	slog.Info("server stopped gracefully")

	return nil
}

func healthHandler(
	w http.ResponseWriter,
	_ *http.Request,
) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readyHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		pingContext, cancel := context.WithTimeout(
			r.Context(),
			2*time.Second,
		)
		defer cancel()

		if err := pool.Ping(pingContext); err != nil {
			slog.Error("readiness database check", "error", err)
			httpx.WriteError(w, http.StatusServiceUnavailable, "not ready")

			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
