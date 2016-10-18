package main

import (
	"flag"
	"github.com/Sirupsen/logrus"
	"net/http"
	"os"

	"github.com/symbiont-io/assembly-sdk/api/rest"
	"github.com/symbiont-io/assembly-sdk/mock"
)

var listen = flag.String("listen", "localhost:4000", "address to listen on")

func newLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Level = logrus.DebugLevel

	fmt := new(logrus.TextFormatter)
	fmt.TimestampFormat = "2006-01-02 15:04:05.000"
	fmt.FullTimestamp = true
	logger.Formatter = fmt
	return logger
}

func main() {
	flag.Parse()

	logger := newLogger()
	s := rest.NewServer(mock.NewLedger(), rest.WithLogger(logger))

	logger.Println("Listening on", *listen)
	logger.Println(http.ListenAndServe(*listen, s.Router()))
}
