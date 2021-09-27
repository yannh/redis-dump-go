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

type progressLogger struct {
	stats map[uint8]int
}

func newProgressLogger() *progressLogger {
	return &progressLogger{
		stats: map[uint8]int{},
	}
}

func (p *progressLogger) drawProgress(to io.Writer, db uint8, nDumped int) {
	if _, ok := p.stats[db]; !ok && len(p.stats) > 0 {
		// We switched database, write to a new line
		fmt.Fprintf(to, "\n")
	}

	p.stats[db] = nDumped
	if nDumped == 0 {
		return
	}

	fmt.Fprintf(to, "\rDatabase %d: %d element dumped", db, nDumped)
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

	host := flag.String("host", "127.0.0.1", "Server host")
	port := flag.Int("port", 6379, "Server port")
	db := flag.Uint("db", 0, "only dump this database (default: all databases)")
	filter := flag.String("filter", "*", "Key filter to use")
	noscan := flag.Bool("noscan", false, "Use KEYS * instead of SCAN - for Redis <=2.8")
	batchSize := flag.Int("batchSize", 1000, "HSET/RPUSH/SADD/ZADD only add 'batchSize' items at a time")
	nWorkers := flag.Int("n", 10, "Parallel workers")
	withTTL := flag.Bool("ttl", true, "Preserve Keys TTL")
	output := flag.String("output", "resp", "Output type - can be resp or commands")
	silent := flag.Bool("s", false, "Silent mode (disable logging of progress / stats)")
	tls := flag.Bool("tls", false, "tls mode (default disable for enabling pass the parameter)")
	caCert := flag.String("cacert", "", "provide cacert file path only if tls enable")
	cert := flag.String("cert", "", "provide cert file path only if tls enable")
	key := flag.String("key", "", "provide key file path only if tls enable")
	flag.Parse()

	if !isFlagPassed("db") {
		db = nil
	}
	var tlshandler *redisdump.TlsHandler
	if isFlagPassed("tls") {
		*tls = true
		tlshandler = redisdump.NewTlsHandler(*tls, *caCert, *cert, *key)
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

	pl := newProgressLogger()
	go func() {
		for n := range progressNotifs {
			if !(*silent) {
				pl.drawProgress(os.Stderr, n.Db, n.Done)
			}
		}
		wg.Done()
	}()
	logger := log.New(os.Stdout, "", 0)
	if db == nil {
		if err = redisdump.DumpServer(*host, *port, redisPassword, tlshandler, *filter, *nWorkers, *withTTL, *batchSize, *noscan, logger, serializer, progressNotifs); err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			return 1
		}
	} else {
		if err = redisdump.DumpDB(*host, *port, redisPassword, uint8(*db), tlshandler, *filter, *nWorkers, *withTTL, *batchSize, *noscan, logger, serializer, progressNotifs); err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			return 1
		}
	}
	return 0
}

func main() {
	os.Exit(realMain())
}
