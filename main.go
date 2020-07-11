package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/yannh/redis-dump-go/redisdump"
)

func drawProgressBar(to io.Writer, currentPosition, nElements, widgetSize int) {
	if nElements == 0 {
		return
	}
	percent := currentPosition * 100 / nElements
	nBars := widgetSize * percent / 100

	bars := strings.Repeat("=", nBars)
	spaces := strings.Repeat(" ", widgetSize-nBars)
	fmt.Fprintf(to, "\r[%s%s] %3d%% [%d/%d]", bars, spaces, int(percent), currentPosition, nElements)
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func realMain() int {
	var err error

	// TODO: Number of workers & TTL as parameters
	host := flag.String("host", "127.0.0.1", "Server host")
	port := flag.Int("port", 6379, "Server port")
	db := flag.Int("db", 0, "only dump this database (default: all databases)")
	nWorkers := flag.Int("n", 10, "Parallel workers")
	withTTL := flag.Bool("ttl", true, "Preserve Keys TTL")
	output := flag.String("output", "resp", "Output type - can be resp or commands")
	silent := flag.Bool("s", false, "Silent mode (disable progress bar)")
	flag.Parse()

	if !isFlagPassed("db") {
		db = nil
	}

	var serializer func([]string) string
	switch *output {
	case "resp":
		serializer = redisdump.RESPSerializer

	case "commands":
		serializer = redisdump.RedisCmdSerializer

	default:
		log.Fatalf("Failed parsing parameter flag: can only be resp or json")
	}

	redisPassword := os.Getenv("REDISDUMPGO_AUTH")

	progressNotifs := make(chan redisdump.ProgressNotification)
	var wg sync.WaitGroup
	wg.Add(1)

	defer func() {
		close(progressNotifs)
		wg.Wait()
		if !(*silent) {
			fmt.Fprint(os.Stderr, "\n")
		}
	}()

	go func() {
		for n := range progressNotifs {
			if !(*silent) {
				drawProgressBar(os.Stderr, n.Done, n.Total, 50)
			}
		}
		wg.Done()
	}()

	logger := log.New(os.Stdout, "", 0)
	if db == nil {
		if err = redisdump.DumpServer(*host, *port, redisPassword, *nWorkers, *withTTL, logger, serializer, progressNotifs); err != nil {
			fmt.Println(err)
			return 1
		}
	} else {
		url := redisdump.RedisURL(*host, fmt.Sprint(*port), fmt.Sprint(*db), redisPassword)
		if err = redisdump.DumpDB(url, *nWorkers, *withTTL, logger, serializer, progressNotifs); err != nil {
			fmt.Println(err)
			return 1
		}
	}
	return 0
}

func main() {
	os.Exit(realMain())
}
