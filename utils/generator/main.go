package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"

	"github.com/yannh/redis-dump-go/pkg/redisdump"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func GenerateStrings(w io.Writer, nKeys int, serializer redisdump.Serializer) {
	for i := 0; i < nKeys; i++ {
		io.WriteString(w, serializer([]string{"SET", randSeq(8), randSeq(16)})+"\n")
	}
}

func GenerateZSET(w io.Writer, nKeys int, serializer redisdump.Serializer) {
	zsetKey := randSeq(16)
	for i := 0; i < nKeys; i++ {
		io.WriteString(w, serializer([]string{"ZADD", zsetKey, "1", randSeq(16)})+"\n")
	}
}

func main() {
	nKeys := flag.Int("n", 100, "Number of keys to generate")
	sType := flag.String("type", "strings", "zset or strings")
	oType := flag.String("output", "resp", "resp or commands")
	flag.Parse()

	var s redisdump.Serializer
	switch strings.ToLower(*oType) {
	case "resp":
		s = redisdump.RESPSerializer

	case "commands":
		s = redisdump.RedisCmdSerializer

	default:
		fmt.Fprintf(os.Stderr, "Unrecognised type %s, should be strings or zset", *sType)
		os.Exit(1)
	}

	switch strings.ToLower(*sType) {
	case "zset":
		GenerateZSET(os.Stdout, *nKeys, s)

	case "strings":
		GenerateStrings(os.Stdout, *nKeys, s)

	default:
		fmt.Fprintf(os.Stderr, "Unrecognised type %s, should be strings or zset", *sType)
		os.Exit(1)
	}
}
