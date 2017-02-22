package main

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	// subscribe to SIGINT signals
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)

	srv := &http.Server{Addr: ":8080", Handler: http.DefaultServeMux}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	go func() {
		<-quit

		d := time.Now().Add(5 * time.Second) // deadline 5s max
		ctx, cancel := context.WithDeadline(context.Background(), d)

		defer cancel()

		log.Println("Shutting down server...")
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("could not shutdown: %v", err)
		}
	}()

	err := srv.ListenAndServe()

	if err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}

	log.Println("Server gracefully stopped")
}
