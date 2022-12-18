package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nexusriot/antiworld/internal/app"
)

func main() {

	rand.Seed(time.Now().UTC().UnixNano())

	currentTime := time.Now()
	logFileName := fmt.Sprintf("job_%s.log", currentTime.Format("2006-01-02-15-04-05"))

	f, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic("error creating log file")
	}
	defer f.Close()
	log.SetOutput(f)
	log.SetLevel(log.InfoLevel)

	var (
		daemonMode bool
		showHelp   bool
	)

	flag.BoolVar(&daemonMode, "d", false, "run as a daemon")
	flag.BoolVar(&showHelp, "h", false, "show help")
	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	a := app.NewApp()
	a.Start(daemonMode)
}
