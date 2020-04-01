package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mtraver/gaelog"
)

// wrappedHandler must be wrapped using gaelog.Wrap or gaelog.WrapWithID so that the
// request context can be used with the package-level logging functions.
type wrappedHandler struct{}

func (h wrappedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()

	gaelog.Debugf(ctx, "Debug")
	gaelog.Infof(ctx, "Info")
	gaelog.Noticef(ctx, "Notice")
	gaelog.Warningf(ctx, "Warning")
	gaelog.Errorf(ctx, "Error")
	gaelog.Criticalf(ctx, "Critical")
	gaelog.Alertf(ctx, "Alert")
	gaelog.Emergencyf(ctx, "Emergency")

	message := struct {
		Places []string
	}{
		[]string{"Kings Canyon", "Sequoia", "Yosemite", "Death Valley"},
	}

	gaelog.Info(ctx, message)

	fmt.Fprintf(w, "Hello!")
}

// manualHandler creates and closes a logger manually. This usage does not require
// gaelog.Wrap or gaelog.WrapWithID.
func manualHandler(w http.ResponseWriter, r *http.Request) {
	lg, err := gaelog.New(r)
	if err != nil {
		// The returned logger is valid despite the error. It falls back to logging
		// via the standard library's "log" package.
		lg.Errorf("Failed to make logger: %v", err)
	}
	defer lg.Close()

	lg.Warningf("Some important info right here, that's for sure")

	fmt.Fprintf(w, "Hello!")
}

func main() {
	// Wrap the handler.
	http.Handle("/", gaelog.Wrap(wrappedHandler{}))

	http.HandleFunc("/manual", manualHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
