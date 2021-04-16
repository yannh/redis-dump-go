package redisdump

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	radix "github.com/mediocregopher/radix/v3"
)

func ttlToRedisCmd(k string, val int64) []string {
	return []string{"EXPIREAT", k, fmt.Sprint(time.Now().Unix() + val)}
}

func stringToRedisCmd(k, val string) []string {
	return []string{"SET", k, val}
}

func hashToRedisCmd(k string, val map[string]string) []string {
	cmd := []string{"HSET", k}
	for k, v := range val {
		cmd = append(cmd, k, v)
	}
	return cmd
}

func setToRedisCmd(k string, val []string) []string {
	cmd := []string{"SADD", k}
	return append(cmd, val...)
}

func listToRedisCmd(k string, val []string) []string {
	cmd := []string{"RPUSH", k}
	return append(cmd, val...)
}

func zsetToRedisCmd(k string, val []string) []string {
	cmd := []string{"ZADD", k}
	var key string

	for i, v := range val {
		if i%2 == 0 {
			key = v
			continue
		}

		cmd = append(cmd, v, key)
	}
	return cmd
}

// RESPSerializer will serialize cmd to RESP
func RESPSerializer(cmd []string) string {
	buf := strings.Builder{}
	buf.WriteString("*" + strconv.Itoa(len(cmd)) + "\r\n")
	for _, arg := range cmd {
		buf.WriteString("$" + strconv.Itoa(len(arg)) + "\r\n" + arg + "\r\n")
	}
	return buf.String()
}

// RedisCmdSerializer will serialize cmd to a string with redis commands
func RedisCmdSerializer(cmd []string) string {
	if len(cmd) == 0 {
		return ""
	}

	buf := strings.Builder{}
	buf.WriteString(fmt.Sprintf("%s", cmd[0]))
	for i:=1;i< len(cmd);i++{
		if strings.Contains(cmd[i], " ") {
			buf.WriteString(fmt.Sprintf(" \"%s\"", cmd[i]))
		} else {
			buf.WriteString(fmt.Sprintf(" %s", cmd[i]))
		}
	}

	return buf.String()
}

func dumpKeys(client radix.Client, keys []string, withTTL bool, logger *log.Logger, serializer func([]string) string) error {
	var err error
	var redisCmd []string

	for _, key := range keys {
		var keyType string

		err = client.Do(radix.Cmd(&keyType, "TYPE", key))
		if err != nil {
			return err
		}

		switch keyType {
		case "string":
			var val string
			if err = client.Do(radix.Cmd(&val, "GET", key)); err != nil {
				return err
			}
			redisCmd = stringToRedisCmd(key, val)

		case "list":
			var val []string
			if err = client.Do(radix.Cmd(&val, "LRANGE", key, "0", "-1")); err != nil {
				return err
			}
			redisCmd = listToRedisCmd(key, val)

		case "set":
			var val []string
			if err = client.Do(radix.Cmd(&val, "SMEMBERS", key)); err != nil {
				return err
			}
			redisCmd = setToRedisCmd(key, val)

		case "hash":
			var val map[string]string
			if err = client.Do(radix.Cmd(&val, "HGETALL", key)); err != nil {
				return err
			}
			redisCmd = hashToRedisCmd(key, val)

		case "zset":
			var val []string
			if err = client.Do(radix.Cmd(&val, "ZRANGEBYSCORE", key, "-inf", "+inf", "WITHSCORES")); err != nil {
				return err
			}
			redisCmd = zsetToRedisCmd(key, val)

		case "none":

		default:
			return fmt.Errorf("Key %s is of unreconized type %s", key, keyType)
		}

		logger.Print(serializer(redisCmd))

		if withTTL {
			var ttl int64
			if err = client.Do(radix.Cmd(&ttl, "TTL", key)); err != nil {
				return err
			}
			if ttl > 0 {
				redisCmd = ttlToRedisCmd(key, ttl)
				logger.Print(serializer(redisCmd))
			}
		}
	}

	return nil
}

func dumpKeysWorker(client radix.Client, keyBatches <-chan []string, withTTL bool, logger *log.Logger, serializer func([]string) string, errors chan<- error, done chan<- bool) {
	for keyBatch := range keyBatches {
		if err := dumpKeys(client, keyBatch, withTTL, logger, serializer); err != nil {
			errors <- err
		}
	}
	done <- true
}

// ProgressNotification message indicates the progress in dumping the Redis server,
// and can be used to provide a progress visualisation such as a progress bar.
// Done is the number of items dumped, Total is the total number of items to dump.
type ProgressNotification struct {
	Db   uint8
	Done int
}

func parseKeyspaceInfo(keyspaceInfo string) ([]uint8, error) {
	var dbs []uint8

	scanner := bufio.NewScanner(strings.NewReader(keyspaceInfo))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if !strings.HasPrefix(line, "db") {
			continue
		}

		dbIndexString := line[2:strings.IndexAny(line, ":")]
		dbIndex, err := strconv.ParseUint(dbIndexString, 10, 8)
		if err != nil {
			return nil, err
		}
		if dbIndex > 16 {
			return nil, fmt.Errorf("Error parsing INFO keyspace")
		}

		dbs = append(dbs, uint8(dbIndex))
	}

	return dbs, nil
}

func getDBIndexes(redisURL string) ([]uint8, error) {
	client, err := radix.NewPool("tcp", redisURL, 1)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var keyspaceInfo string
	if err = client.Do(radix.Cmd(&keyspaceInfo, "INFO", "keyspace")); err != nil {
		return nil, err
	}

	return parseKeyspaceInfo(keyspaceInfo)
}

func scanKeys(client radix.Client, db uint8, filter string, keyBatches chan<- []string, progressNotifications chan<- ProgressNotification) error {
	keyBatchSize := 100
	s := radix.NewScanner(client, radix.ScanOpts{Command: "SCAN", Pattern: filter, Count: keyBatchSize})

	nProcessed := 0
	var key string
	var keyBatch []string
	for s.Next(&key) {
		keyBatch = append(keyBatch, key)
		if len(keyBatch) >= keyBatchSize {
			nProcessed += len(keyBatch)
			keyBatches <- keyBatch
			keyBatch = nil
			progressNotifications <- ProgressNotification{Db: db, Done: nProcessed}
		}
	}

	keyBatches <- keyBatch
	nProcessed += len(keyBatch)
	progressNotifications <- ProgressNotification{Db: db, Done: nProcessed}

	return s.Close()
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func scanKeysLegacy(client radix.Client, db uint8, filter string, keyBatches chan<- []string, progressNotifications chan<- ProgressNotification) error {
	keyBatchSize := 100
	var err error
	var keys []string
	if err = client.Do(radix.Cmd(&keys, "KEYS", filter)); err != nil {
		return err
	}

	for i := 0; i < len(keys); i += keyBatchSize {
		batchEnd := min(i+keyBatchSize, len(keys))
		keyBatches <- keys[i:batchEnd]
		if progressNotifications != nil {
			progressNotifications <- ProgressNotification{db, i}
		}
	}

	return nil
}

// RedisURL builds a connect URL given a Host, port, db & password
func RedisURL(redisHost string, redisPort string, redisDB string, redisPassword string) string {
	switch {
	case redisDB == "":
		return "redis://:" + redisPassword + "@" + redisHost + ":" + fmt.Sprint(redisPort)
	case redisDB != "":
		return "redis://:" + redisPassword + "@" + redisHost + ":" + fmt.Sprint(redisPort) + "/" + redisDB
	}

	return ""
}

// DumpDB dumps all keys from a single Redis DB
func DumpDB(redisHost string, redisPort int, redisPassword string, db uint8, filter string, nWorkers int, withTTL bool, noscan bool, logger *log.Logger, serializer func([]string) string, progress chan<- ProgressNotification) error {
	var err error

	keyGenerator := scanKeys
	if noscan {
		keyGenerator = scanKeysLegacy
	}

	errors := make(chan error)
	nErrors := 0
	go func() {
		for err := range errors {
			fmt.Fprintln(os.Stderr, "Error: "+err.Error())
			nErrors++
		}
	}()

	redisURL := RedisURL(redisHost, fmt.Sprint(redisPort), fmt.Sprint(db), redisPassword)

	customConnFunc := func(network, addr string) (radix.Conn, error) {
		return radix.Dial(network, addr,
			radix.DialTimeout(5 * time.Minute),
		)
	}

	client, err := radix.NewPool("tcp", redisURL, nWorkers, radix.PoolConnFunc(customConnFunc))
	if err != nil {
		return err
	}
	defer client.Close()

	if err = client.Do(radix.Cmd(nil, "SELECT", fmt.Sprint(db))); err != nil {
		return err
	}
	logger.Printf(serializer([]string{"SELECT", fmt.Sprint(db)}))

	done := make(chan bool)
	keyBatches := make(chan []string)
	for i := 0; i < nWorkers; i++ {
		go dumpKeysWorker(client, keyBatches, withTTL, logger, serializer, errors, done)
	}

	keyGenerator(client, db, filter, keyBatches, progress)
	close(keyBatches)

	for i := 0; i < nWorkers; i++ {
		<-done
	}

	return nil
}

// DumpServer dumps all Keys from the redis server given by redisURL,
// to the Logger logger. Progress notification informations
// are regularly sent to the channel progressNotifications
func DumpServer(redisHost string, redisPort int, redisPassword string, filter string, nWorkers int, withTTL bool, noscan bool, logger *log.Logger, serializer func([]string) string, progress chan<- ProgressNotification) error {
	url := RedisURL(redisHost, fmt.Sprint(redisPort), "", redisPassword)
	dbs, err := getDBIndexes(url)
	if err != nil {
		return err
	}

	for _, db := range dbs {
		if err = DumpDB(redisHost, redisPort, redisPassword, db, filter, nWorkers, withTTL, noscan, logger, serializer, progress); err != nil {
			return err
		}
	}

	return nil
}
