package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
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

type Config struct {
	host      string
	port      int
	db        *uint
	filter    string
	noscan    bool
	batchSize int
	nWorkers  int
	withTTL   bool
	output    string
	silent    bool
	tls       bool
	caCert    string
	cert      string
	key       string
}

func FromFlags(progName string, args []string) (Config, error) {
	c := Config{}

	flags := flag.NewFlagSet(progName, flag.ExitOnError)
	var buf bytes.Buffer
	flags.SetOutput(&buf)

	flags.StringVar(&c.host, "host", "127.0.0.1", "Server host")
	flags.IntVar(&c.port, "port", 6379, "Server port")
	flags.UintVar(c.db, "db", 0, "only dump this database (default: all databases)")
	flags.StringVar(&c.filter, "filter", "*", "Key filter to use")
	flags.BoolVar(&c.noscan, "noscan", false, "Use KEYS * instead of SCAN - for Redis <=2.8")
	flags.IntVar(&c.batchSize, "batchSize", 1000, "HSET/RPUSH/SADD/ZADD only add 'batchSize' items at a time")
	flags.IntVar(&c.nWorkers, "n", 10, "Parallel workers")
	flags.BoolVar(&c.withTTL, "ttl", true, "Preserve Keys TTL")
	flags.StringVar(&c.output, "output", "resp", "Output type - can be resp or commands")
	flags.BoolVar(&c.silent, "s", false, "Silent mode (disable logging of progress / stats)")
	flags.BoolVar(&c.tls, "tls", false, "Establish a secure TLS connection")
	flags.StringVar(&c.caCert, "cacert", "", "CA Certificate file to verify with")
	flags.StringVar(&c.cert, "cert", "", "Private key file to authenticate with")
	flags.StringVar(&c.key, "key", "", "SSL private key file path")
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]... [FILE OR FOLDER]...\n", progName)

		flags.SetOutput(os.Stderr)
		flags.PrintDefaults()
	}

	if !isFlagPassed("db") {
		c.db = nil
	}

	err := flags.Parse(args)
	return c, err
}

func realMain() int {
	var err error

	c, _ := FromFlags(os.Args[0], os.Args[1:])

	var tlshandler *redisdump.TlsHandler = nil
	if c.tls == true {
		tlshandler = redisdump.NewTlsHandler(c.caCert, c.cert, c.key)
	}

	var serializer func([]string) string
	switch c.output {
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
		if !(c.silent) {
			fmt.Fprint(os.Stderr, "\n")
		}
	}()

	pl := newProgressLogger()
	go func() {
		for n := range progressNotifs {
			if !(c.silent) {
				pl.drawProgress(os.Stderr, n.Db, n.Done)
			}
		}
		wg.Done()
	}()

	logger := log.New(os.Stdout, "", 0)
	if c.db == nil {
		if err = redisdump.DumpServer(c.host, c.port, url.QueryEscape(redisPassword), tlshandler, c.filter, c.nWorkers, c.withTTL, c.batchSize, c.noscan, logger, serializer, progressNotifs); err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			return 1
		}
	} else {
		if err = redisdump.DumpDB(c.host, c.port, url.QueryEscape(redisPassword), tlshandler, uint8(*c.db), c.filter, c.nWorkers, c.withTTL, c.batchSize, c.noscan, logger, serializer, progressNotifs); err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			return 1
		}
	}
	return 0
}

func main() {
	os.Exit(realMain())
}
