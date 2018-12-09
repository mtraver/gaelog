# Easy Stackdriver Logging on Google App Engine Standard second generation runtimes

[![GoDoc](https://godoc.org/github.com/mtraver/gaelog?status.svg)](https://godoc.org/github.com/mtraver/gaelog)
[![Go Report Card](https://goreportcard.com/badge/github.com/mtraver/gaelog)](https://goreportcard.com/report/github.com/mtraver/gaelog)

Using Stackdriver Logging on App Engine Standard is complicated. It doesn't
have to be that way.

```go
package main

import (
  "fmt"
  "log"
  "net/http"
  "os"

  "github.com/mtraver/gaelog"
)

func index(w http.ResponseWriter, r *http.Request) {
  lg, err := gaelog.New(r)
  if err != nil {
    // The returned logger is valid despite the error. It falls back to logging
    // via the standard library's "log" package.
    lg.Errorf("Failed to make logger: %v", err)
  }
  defer lg.Close()

  lg.Debugf("Debug")
  lg.Infof("Info")
  lg.Noticef("Notice")
  lg.Warningf("Warning")
  lg.Errorf("Error")
  lg.Criticalf("Critical")
  lg.Alertf("Alert")
  lg.Emergencyf("Emergency")

  message := struct {
    Places []string
  }{
    []string{"Kings Canyon", "Sequoia", "Yosemite", "Death Valley"},
  }

  lg.Info(message)
}

func main() {
  http.HandleFunc("/", index)

  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
  }
  log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
```

![Screenshot of logs in Stackdriver UI](images/log_levels.png)
