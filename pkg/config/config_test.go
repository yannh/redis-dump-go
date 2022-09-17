package config

import (
	"reflect"
	"testing"
)

func TestFromFlags(t *testing.T) {
	testCases := []struct {
		args []string
		conf Config
	}{
		{
			[]string{},
			Config{
				Db:        -1,
				Host:      "127.0.0.1",
				Port:      6379,
				Filter:    "*",
				BatchSize: 1000,
				NWorkers:  10,
				WithTTL:   true,
				Output:    "resp",
				Insecure:  false,
			},
		},
		{
			[]string{"-db", "2"},
			Config{
				Db:        2,
				Host:      "127.0.0.1",
				Port:      6379,
				Filter:    "*",
				BatchSize: 1000,
				NWorkers:  10,
				WithTTL:   true,
				Output:    "resp",
				Insecure:  false,
			},
		},
		{
			[]string{"-ttl=false"},
			Config{
				Db:        -1,
				Host:      "127.0.0.1",
				Port:      6379,
				Filter:    "*",
				BatchSize: 1000,
				NWorkers:  10,
				WithTTL:   false,
				Output:    "resp",
				Insecure:  false,
			},
		},
		{
			[]string{"-host", "redis", "-port", "1234", "-batchSize", "10", "-n", "5", "-output", "commands"},
			Config{
				Db:        -1,
				Host:      "redis",
				Port:      1234,
				Filter:    "*",
				BatchSize: 10,
				NWorkers:  5,
				WithTTL:   true,
				Output:    "commands",
				Insecure:  false,
			},
		},
		{
			[]string{"-host", "redis", "-port", "1234", "-batchSize", "10", "-user", "test", "-insecure"},
			Config{
				Db:        -1,
				Host:      "redis",
				Port:      1234,
				Filter:    "*",
				BatchSize: 10,
				NWorkers:  10,
				WithTTL:   true,
				Output:    "resp",
				Username:  "test",
				Insecure:  true,
			},
		},
		{
			[]string{"-host", "redis", "-port", "1234", "-batchSize", "10", "-user", "test"},
			Config{
				Db:        -1,
				Host:      "redis",
				Port:      1234,
				Filter:    "*",
				BatchSize: 10,
				NWorkers:  10,
				WithTTL:   true,
				Output:    "resp",
				Username:  "test",
			},
		},
		{
			[]string{"-db", "1"},
			Config{
				Db:        1,
				Host:      "127.0.0.1",
				Port:      6379,
				Filter:    "*",
				BatchSize: 1000,
				NWorkers:  10,
				WithTTL:   true,
				Output:    "resp",
				Insecure:  false,
			},
		},
		{
			[]string{"-h"},
			Config{
				Db:        -1,
				Host:      "127.0.0.1",
				Port:      6379,
				Filter:    "*",
				BatchSize: 1000,
				NWorkers:  10,
				WithTTL:   true,
				Output:    "resp",
				Help:      true,
				Insecure:  false,
			},
		},
	}

	for i, testCase := range testCases {
		cfg, _, _ := FromFlags("redis-dump-go", testCase.args)
		if reflect.DeepEqual(cfg, testCase.conf) != true {
			t.Errorf("test %d: failed parsing config - expected , got: \n%+v\n%+v", i, testCase.conf, cfg)
		}
	}
}
