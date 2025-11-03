package config

import (
	"bytes"
	"flag"
	"fmt"
	"strings"
)

// stringSlice implements flag.Value interface to support multiple flag values
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type Config struct {
	Host           string
	Port           int
	Db             int
	Username       string
	Filter         string
	SkipFilters    []string
	Noscan         bool
	BatchSize      int
	NWorkers       int
	WithTTL        bool
	MinRandomTTL   int
	MaxRandomTTL   int
	Output         string
	Silent         bool
	Tls            bool
	Insecure       bool
	CaCert         string
	Cert           string
	Key            string
	Help           bool
}

func isFlagPassed(flags *flag.FlagSet, name string) bool {
	found := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func FromFlags(progName string, args []string) (Config, string, error) {
	c := Config{}
	var skipFilters stringSlice

	flags := flag.NewFlagSet(progName, flag.ContinueOnError)
	var outBuf bytes.Buffer
	flags.SetOutput(&outBuf)

	flags.StringVar(&c.Host, "host", "127.0.0.1", "Server host")
	flags.IntVar(&c.Port, "port", 6379, "Server port")
	flags.IntVar(&c.Db, "db", -1, "only dump this database (default: all databases)")
	flags.StringVar(&c.Username, "user", "", "Username")
	flags.StringVar(&c.Filter, "filter", "*", "Key filter to use")
	flags.Var(&skipFilters, "skip-filter", "Key filter to exclude (can be specified multiple times)")
	flags.BoolVar(&c.Noscan, "noscan", false, "Use KEYS * instead of SCAN - for Redis <=2.8")
	flags.IntVar(&c.BatchSize, "batchSize", 1000, "HSET/RPUSH/SADD/ZADD only add 'batchSize' items at a time")
	flags.IntVar(&c.NWorkers, "n", 10, "Parallel workers")
	flags.BoolVar(&c.WithTTL, "ttl", true, "Preserve Keys TTL")
	flags.IntVar(&c.MinRandomTTL, "min-random-ttl", 0, "Minimum random TTL in seconds (when > 0, TTL queries are skipped)")
	flags.IntVar(&c.MaxRandomTTL, "max-random-ttl", 0, "Maximum random TTL in seconds (when > 0, TTL queries are skipped)")
	flags.StringVar(&c.Output, "output", "resp", "Output type - can be resp or commands")
	flags.BoolVar(&c.Silent, "s", false, "Silent mode (disable logging of progress / stats)")
	flags.BoolVar(&c.Tls, "tls", false, "Establish a secure TLS connection")
	flags.BoolVar(&c.Insecure, "insecure", false, "Allow insecure TLS connection by skipping cert validation")
	flags.StringVar(&c.CaCert, "cacert", "", "CA Certificate file to verify with")
	flags.StringVar(&c.Cert, "cert", "", "Private key file to authenticate with")
	flags.StringVar(&c.Key, "key", "", "SSL private key file path")
	flags.BoolVar(&c.Help, "h", false, "show help information")
	flags.Usage = func() {
		fmt.Fprintf(&outBuf, "Usage: %s [OPTION]...\n", progName)
		flags.PrintDefaults()
	}

	err := flags.Parse(args)

	if c.Help {
		flags.Usage()
	}

	c.SkipFilters = skipFilters

	return c, outBuf.String(), err
}
