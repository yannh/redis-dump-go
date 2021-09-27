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

func hashToRedisCmds(hashKey string, val map[string]string, batchSize int) [][]string {
	cmds := [][]string{}

	cmd := []string{"HSET", hashKey}
	n := 0
	for k, v := range val {
		if n >= batchSize {
			n = 0
			cmds = append(cmds, cmd)
			cmd = []string{"HSET", hashKey}
		}
		cmd = append(cmd, k, v)
		n++
	}

	if n > 0 {
		cmds = append(cmds, cmd)
	}

	return cmds
}

func setToRedisCmds(setKey string, val []string, batchSize int) [][]string {
	cmds := [][]string{}
	cmd := []string{"SADD", setKey}
	n := 0
	for _, v := range val {
		if n >= batchSize {
			n = 0
			cmds = append(cmds, cmd)
			cmd = []string{"SADD", setKey}
		}
		cmd = append(cmd, v)
		n++
	}

	if n > 0 {
		cmds = append(cmds, cmd)
	}

	return cmds
}

func listToRedisCmds(listKey string, val []string, batchSize int) [][]string {
	cmds := [][]string{}
	cmd := []string{"RPUSH", listKey}
	n := 0
	for _, v := range val {
		if n >= batchSize {
			n = 0
			cmds = append(cmds, cmd)
			cmd = []string{"RPUSH", listKey}
		}
		cmd = append(cmd, v)
		n++
	}

	if n > 0 {
		cmds = append(cmds, cmd)
	}

	return cmds
}

// We break down large ZSETs to multiple ZADD commands

func zsetToRedisCmds(zsetKey string, val []string, batchSize int) [][]string {
	cmds := [][]string{}
	var key string

	cmd := []string{"ZADD", zsetKey}
	n := 0
	for i, v := range val {
		if i%2 == 0 {
			key = v
			continue
		}

		if n >= batchSize {
			n = 0
			cmds = append(cmds, cmd)
			cmd = []string{"ZADD", zsetKey}
		}
		cmd = append(cmd, v, key)
		n++
	}

	if n > 0 {
		cmds = append(cmds, cmd)
	}

	return cmds
}

type Serializer func([]string) string

// RedisCmdSerializer will serialize cmd to a string with redis commands
func RedisCmdSerializer(cmd []string) string {
	if len(cmd) == 0 {
		return ""
	}

	buf := strings.Builder{}
	buf.WriteString(fmt.Sprintf("%s", cmd[0]))
	for i := 1; i < len(cmd); i++ {
		if strings.Contains(cmd[i], " ") {
			buf.WriteString(fmt.Sprintf(" \"%s\"", cmd[i]))
		} else {
			buf.WriteString(fmt.Sprintf(" %s", cmd[i]))
		}
	}

	return buf.String()
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

func dumpKeys(client radix.Client, keys []string, withTTL bool, batchsize int, logger *log.Logger, serializer Serializer) error {
	var err error
	var redisCmds [][]string

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
			redisCmds = [][]string{stringToRedisCmd(key, val)}

		case "list":
			var val []string
			if err = client.Do(radix.Cmd(&val, "LRANGE", key, "0", "-1")); err != nil {
				return err
			}
			redisCmds = listToRedisCmds(key, val, batchsize)

		case "set":
			var val []string
			if err = client.Do(radix.Cmd(&val, "SMEMBERS", key)); err != nil {
				return err
			}
			redisCmds = setToRedisCmds(key, val, batchsize)

		case "hash":
			var val map[string]string
			if err = client.Do(radix.Cmd(&val, "HGETALL", key)); err != nil {
				return err
			}
			redisCmds = hashToRedisCmds(key, val, batchsize)

		case "zset":
			var val []string
			if err = client.Do(radix.Cmd(&val, "ZRANGEBYSCORE", key, "-inf", "+inf", "WITHSCORES")); err != nil {
				return err
			}
			redisCmds = zsetToRedisCmds(key, val, batchsize)

		case "none":

		default:
			return fmt.Errorf("Key %s is of unreconized type %s", key, keyType)
		}

		for _, redisCmd := range redisCmds {
			logger.Print(serializer(redisCmd))
		}

		if withTTL {
			var ttl int64
			if err = client.Do(radix.Cmd(&ttl, "TTL", key)); err != nil {
				return err
			}
			if ttl > 0 {
				cmd := ttlToRedisCmd(key, ttl)
				logger.Print(serializer(cmd))
			}
		}
	}

	return nil
}

func dumpKeysWorker(client radix.Client, keyBatches <-chan []string, withTTL bool, batchSize int, logger *log.Logger, serializer Serializer, errors chan<- error, done chan<- bool) {
	for keyBatch := range keyBatches {
		if err := dumpKeys(client, keyBatch, withTTL, batchSize, logger, serializer); err != nil {
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

func getDBIndexes(redisURL string, redisPassword string, tlsHandler *TlsHandler) ([]uint8, error) {

	client, err := NewRedisClient(redisURL, tlsHandler, redisPassword, 1, "")
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

// RedisURL builds a connect URL given a Host, port
func RedisURL(redisHost string, redisPort string) string {
	return redisHost + ":" + fmt.Sprint(redisPort)
}

// DumpDB dumps all keys from a single Redis DB
func DumpDB(redisHost string, redisPort int, redisPassword string, db uint8, tlsHandler *TlsHandler, filter string, nWorkers int, withTTL bool, batchSize int, noscan bool, logger *log.Logger, serializer Serializer, progress chan<- ProgressNotification) error {
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
	redisURL := RedisURL(redisHost, fmt.Sprint(redisPort))
	client, err := NewRedisClient(redisURL, tlsHandler, redisPassword, nWorkers, fmt.Sprint(db))
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
		go dumpKeysWorker(client, keyBatches, withTTL, batchSize, logger, serializer, errors, done)
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
func DumpServer(redisHost string, redisPort int, redisPassword string, tlsHandler *TlsHandler, filter string, nWorkers int, withTTL bool, batchSize int, noscan bool, logger *log.Logger, serializer func([]string) string, progress chan<- ProgressNotification) error {
	url := RedisURL(redisHost, fmt.Sprint(redisPort))
	dbs, err := getDBIndexes(url, redisPassword, tlsHandler)
	if err != nil {
		return err
	}
	for _, db := range dbs {
		if err = DumpDB(redisHost, redisPort, redisPassword, db, tlsHandler, filter, nWorkers, withTTL, batchSize, noscan, logger, serializer, progress); err != nil {
			return err
		}
	}

	return nil
}
