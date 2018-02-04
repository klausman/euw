package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	poll     = flag.Bool("poll", false, "Poll edacutil repeatedly and log errors to syslog")
	delay    = flag.Duration("delay", 10*time.Second, "Time between polls.")
	logZero  = flag.Duration("logzero", 0*time.Second, "How often to log even if counters are 0")
	edacutil = flag.String("edacutil", "edacutil", "Name and of edacutil binary to run")
)

func main() {
	flag.Parse()
	if *poll {
		runForever()
	} else {
		os.Exit(runOnce())
	}
}

func runOnce() int {
	data, err := runEdacUtil()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running edacutil: %s\n", err)
		return 1
	}
	l := log.New(os.Stderr, "", 0)
	totalerrors := parseEdacUtilOutput(data, l)
	if totalerrors != 0 {
		fmt.Fprintf(os.Stderr, "EDAC reports a total of %d errors.\n", totalerrors)
		return 2
	}
	if totalerrors == 0 && *logZero != 0 {
		l.Printf("EDAC reports no errors.\n")
	}
	return 0
}

func runForever() {
	var lastzerolog time.Time
	alertL := mustMakeLogger(syslog.LOG_ALERT|syslog.LOG_KERN, 0)
	errL := mustMakeLogger(syslog.LOG_ALERT|syslog.LOG_KERN, 0)
	infoL := mustMakeLogger(syslog.LOG_INFO|syslog.LOG_KERN, 0)
	if *logZero != 0 {
		infoL.Printf("Polling EDAC every %v, logging every %v even if no errors reported.\n", *delay, *logZero)
	} else {
		infoL.Printf("Polling EDAC every %v, logging only when errors occur.\n", *delay)
	}
	for {
		totalerrors := 0
		data, err := runEdacUtil()
		if err != nil {
			errL.Printf("Error running edacutil: %s\n", err)
			continue
		} else {
			totalerrors = parseEdacUtilOutput(data, alertL)
			if totalerrors != 0 {
				errL.Printf("EDAC reports a total of %d errors.\n", totalerrors)
			} else if *logZero != 0 && time.Now().Sub(lastzerolog) > *logZero {
				infoL.Printf("EDAC reports no errors.\n")
				lastzerolog = time.Now()
			}
		}
		time.Sleep(*delay)
	}
}

func mustMakeLogger(p syslog.Priority, flags int) *log.Logger {
	l, err := syslog.NewLogger(p, flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create logger: %+v\n", err)
		os.Exit(1)
	}
	return l
}

func runEdacUtil() (string, error) {
	var out bytes.Buffer
	cmd := exec.Command("edac-util", "--report=full")
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func parseEdacUtilOutput(data string, l *log.Logger) int {
	totalerrors := 0
	var etype string
	for _, line := range strings.Split(data, "\n") {
		if len(line) == 0 {
			continue
		}
		tokens := strings.Split(line, ":")
		if len(tokens) != 5 {
			l.Printf("Unparsed line in edacutil output: %+v\n", line)
			continue
		}
		count, err := strconv.ParseInt(tokens[4], 0, 0)
		if err != nil {
			l.Printf("Garbled line in edacutil output: %+v\n", line)
			continue
		}
		if count != 0 {
			controller := tokens[0]
			row := tokens[1]
			infostring := tokens[2]
			if tokens[3] == "UE" {
				etype = "uncorrected"
			} else if tokens[3] == "CE" {
				etype = "corrected"
			} else {
				etype = "unknown state"
			}
			l.Printf("EDAC reports %d %s errors on %s (%s/%s)\n",
				count, etype, infostring, controller, row)
			totalerrors += int(count)
		}
	}
	return totalerrors
}
