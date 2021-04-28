package main

import (
	"flag"
	"github.com/yannh/redis-dump-go/redisdump"
	"io"
	"math/rand"
	"os"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func GenerateData(w io.Writer, nKeys int) {
	for i := 0; i < nKeys; i++ {
		io.WriteString(w, redisdump.RESPSerializer([]string{"SET", randSeq(8), randSeq(16)})+"\n")
	}
}

func main() {
	nKeys := flag.Int("n", 100, "Number of keys to generate")
	flag.Parse()
	GenerateData(os.Stdout, *nKeys)
}
