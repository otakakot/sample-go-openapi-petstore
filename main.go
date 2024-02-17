package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//go:embed openapi.yaml
var yaml embed.FS

func main() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	hdl := http.NewServeMux()

	hel := &health{}

	hdl.Handle("GET /", cors(hel))

	hdl.Handle("GET /health", cors(hel))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: hdl,
	}

	go func() {
		slog.Info("start app server http://localhost:" + port)

		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	mux := http.NewServeMux()

	mux.Handle("GET /", cors(http.FileServer(http.FS(yaml))))

	mux.HandleFunc("GET /petstore", func(w http.ResponseWriter, r *http.Request) {
		yaml := "http://localhost:3000/openapi.yaml"

		petstore := "https://petstore3.swagger.io/?url=" + yaml

		http.Redirect(w, r, petstore, http.StatusFound)
	})

	doc := &http.Server{
		Addr:    ":3000",
		Handler: mux,
	}

	go func() {
		slog.Info("start doc server http://localhost:3000")

		if err := doc.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	<-ctx.Done()

	slog.Info("start shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}

	if err := doc.Shutdown(ctx); err != nil {
		panic(err)
	}

	slog.Info("done server shutdown")
}

func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		h.ServeHTTP(w, r)
	})
}

type health struct{}

func (h *health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}
