package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/yannh/redis-dump-go/redisdump"
)

func drawProgress(to io.Writer, nDumped int) {
	if nDumped == 0 {
		return
	}

	fmt.Fprintf(to, "\r%d element dumped", nDumped)
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
	filter := flag.String("filter", "*", "key filter to use")
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
				drawProgress(os.Stderr, n.Done)
			}
		}
		wg.Done()
	}()

	logger := log.New(os.Stdout, "", 0)
	if db == nil {
		if err = redisdump.DumpServer(*host, *port, redisPassword, *filter, *nWorkers, *withTTL, logger, serializer, progressNotifs); err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			return 1
		}
	} else {
		url := redisdump.RedisURL(*host, fmt.Sprint(*port), fmt.Sprint(*db), redisPassword)
		if err = redisdump.DumpDB(url, *filter, *nWorkers, *withTTL, logger, serializer, progressNotifs); err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			return 1
		}
	}
	return 0
}

func main() {
	os.Exit(realMain())
}
