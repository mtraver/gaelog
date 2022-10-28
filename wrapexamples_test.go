package gaelog_test

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mtraver/gaelog"
)

func ExampleWrapWithID() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		gaelog.Warningf(ctx, "Some important info right here, that's for sure")

		fmt.Fprintf(w, "Hey")
	})

	http.Handle("/", gaelog.WrapWithID(handler, "my_log"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func ExampleWrap() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		gaelog.Warningf(ctx, "Some important info right here, that's for sure")

		fmt.Fprintf(w, "Hey")
	})

	http.Handle("/", gaelog.Wrap(handler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
