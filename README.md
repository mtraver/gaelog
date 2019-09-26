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

## Known issues

1. **Request log (aka parent) severity is not set.**
A nice property of `google.golang.org/appengine/log` is that the severity of the request log entry (aka
parent log entry) is set to the maximum severity of the log entries correlated with it. This makes it easy
to see in the Stackdriver UI which requests have logs associated with them, and at which severity. Alas, that
is not possible with this package. App Engine itself makes the request log entries and it does not know about
any entries created separately (such as with this package). Furthermore, entries cannot be modified after they
are created. A possible remedy is for this package to emit request log entries of its own; open an issue if
you'd like this and we can discuss.

1. **If a request has any log entries made by `google.golang.org/appengine/log`, then
entries made by this package will not be correlated (i.e. nested) with the request in
the Stackdriver Logging UI.**
The corollary is that requests that have any log entries emitted by App Engine itself, such as requests
that time out or requests that start up a new instance, are also subject to this limitation. It seems
like a bug in Stackdriver.
