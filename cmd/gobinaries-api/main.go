package main

import (
	"context"
	"net/http"

	googlestorage "cloud.google.com/go/storage"
	"github.com/apex/httplog"
	"github.com/apex/log"
	"github.com/google/go-github/v28/github"
	"github.com/tj/go/env"
	"golang.org/x/oauth2"

	"github.com/tj/gobinaries/resolver"
	"github.com/tj/gobinaries/server"
	"github.com/tj/gobinaries/storage"
)

type NullFlusher struct {
}

func (n NullFlusher) Flush() error {
	return nil
}

// main
func main() {
	// context
	ctx := context.Background()

	// github client
	gh := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: env.Get("GITHUB_TOKEN"),
		},
	)

	// storage client
	gs, err := googlestorage.NewClient(ctx)
	if err != nil {
		log.Fatalf("error creating storage client: %s", err)
	}

	// server
	addr := ":" + env.GetDefault("PORT", "3000")
	s := &server.Server{
		Static: "static",
		URL:    env.GetDefault("URL", "http://127.0.0.1"+addr),
		Resolver: &resolver.GitHub{
			Client: github.NewClient(oauth2.NewClient(ctx, gh)),
		},
		Storage: &storage.Google{
			Client: gs,
			Bucket: "gobinaries",
			Prefix: "production",
		},
	}

	x := NullFlusher{}

	// add request level logging
	h := flusher(httplog.New(s), x)

	// listen
	log.WithField("addr", addr).Info("starting server")
	err = http.ListenAndServe(addr, h)
	if err != nil {
		log.Fatalf("error: %s", err)
	}
}

// Flusher interface.
type Flusher interface {
	Flush() error
}

// flusher returns an HTTP handler which flushes after each request.
func flusher(h http.Handler, f Flusher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)

		err := f.Flush()
		if err != nil {
			log.WithError(err).Error("error flushing logs")
			return
		}
	})
}
