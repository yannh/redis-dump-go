package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/yannh/redis-dump-go/redisdump"
)

func drawProgressBar(to io.Writer, currentPosition, nElements, widgetSize int) {
	percent := currentPosition * 100 / nElements
	nBars := widgetSize * percent / 100

	bars := strings.Repeat("=", nBars)
	spaces := strings.Repeat(" ", widgetSize-nBars)
	fmt.Fprintf(to, "\r[%s%s] %3d%% [%d/%d]", bars, spaces, int(percent), currentPosition, nElements)
}

func realMain() int {
	var err error
	var wg sync.WaitGroup

	host := flag.String("host", "127.0.0.1", "Server host")
	port := flag.Int("port", 6379, "Server port")
	silent := flag.Bool("s", false, "Silent mode (disable progress bar)")
	flag.Parse()

	var progressNotifs chan redisdump.ProgressNotification
	if !(*silent) {
		progressNotifs = make(chan redisdump.ProgressNotification)
		defer func() {
			close(progressNotifs)
			wg.Wait()
			fmt.Fprint(os.Stderr, "\n")
		}()

		wg.Add(1)

		go func() {
			for n := range progressNotifs {
				drawProgressBar(os.Stderr, n.Done, n.Total, 50)
			}
			wg.Done()
		}()
	}

	logger := log.New(os.Stdout, "", 0)
	if err = redisdump.DumpDb(*host+":"+strconv.Itoa(*port), logger, progressNotifs); err != nil {
		fmt.Println(err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(realMain())
}
