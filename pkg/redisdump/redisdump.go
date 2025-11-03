package redisdump

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	radix "github.com/mediocregopher/radix/v3"
)

var AllDBs *uint8 = nil

// generateRandomTTL generates a random TTL between min and max (inclusive)
func generateRandomTTL(min, max int) int64 {
	if min <= 0 || max <= 0 || min > max {
		return 0
	}
	// Use math/rand with time-based seeding for better randomness
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return int64(r.Intn(max-min+1) + min)
}

// shouldSkipKey checks if a key matches any of the skip filter patterns.
// If a pattern is malformed (invalid syntax), it is ignored and won't match any keys.
func shouldSkipKey(key string, skipFilters []string) bool {
	for _, pattern := range skipFilters {
		matched, err := filepath.Match(pattern, key)
		if err == nil && matched {
			return true
		}
		// Silently ignore malformed patterns - they won't skip any keys
	}
	return false
}

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
		if strings.Contains(cmd[i], " ") || len(cmd[i]) == 0 {
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

type radixCmder func(rcv interface{}, cmd string, args ...string) radix.CmdAction

func dumpKeys(client radix.Client, cmd radixCmder, keys []string, skipFilters []string, withTTL bool, minRandomTTL int, maxRandomTTL int, batchSize int, logger *log.Logger, serializer Serializer) error {
	var err error
	var redisCmds [][]string
	
	// Determine if we should use random TTL
	useRandomTTL := minRandomTTL > 0 && maxRandomTTL > 0 && minRandomTTL <= maxRandomTTL

	// First pass: get all key types and separate string keys for batch MGET
	type keyInfo struct {
		key     string
		keyType string
	}
	var keyInfos []keyInfo
	var stringKeys []string
	
	for _, key := range keys {
		// Skip keys that match any skip filter pattern
		if shouldSkipKey(key, skipFilters) {
			continue
		}

		keyType := ""
		err = client.Do(cmd(&keyType, "TYPE", key))
		if err != nil {
			return err
		}
		
		if keyType == "string" {
			stringKeys = append(stringKeys, key)
		}
		keyInfos = append(keyInfos, keyInfo{key: key, keyType: keyType})
	}

	// Batch fetch string values using MGET
	stringValues := make(map[string]string)
	if len(stringKeys) > 0 {
		// Process string keys in batches of up to 1000
		mgetBatchSize := 1000
		for i := 0; i < len(stringKeys); i += mgetBatchSize {
			end := i + mgetBatchSize
			if end > len(stringKeys) {
				end = len(stringKeys)
			}
			batch := stringKeys[i:end]
			
			// MGET command expects variadic args after the command
			var vals []string
			args := make([]string, len(batch))
			copy(args, batch)
			
			if err = client.Do(cmd(&vals, "MGET", args...)); err != nil {
				return err
			}
			
			// Map values to keys
			for j, val := range vals {
				if j < len(batch) {
					stringValues[batch[j]] = val
				}
			}
		}
	}

	// Second pass: process all keys with their values
	for _, ki := range keyInfos {
		switch ki.keyType {
		case "string":
			val, ok := stringValues[ki.key]
			if !ok {
				// Fallback to GET if MGET didn't work
				if err = client.Do(cmd(&val, "GET", ki.key)); err != nil {
					return err
				}
			}
			redisCmds = [][]string{stringToRedisCmd(ki.key, val)}

		case "list":
			var val []string
			if err = client.Do(cmd(&val, "LRANGE", ki.key, "0", "-1")); err != nil {
				return err
			}
			redisCmds = listToRedisCmds(ki.key, val, batchSize)

		case "set":
			var val []string
			if err = client.Do(cmd(&val, "SMEMBERS", ki.key)); err != nil {
				return err
			}
			redisCmds = setToRedisCmds(ki.key, val, batchSize)

		case "hash":
			var val map[string]string
			if err = client.Do(cmd(&val, "HGETALL", ki.key)); err != nil {
				return err
			}
			redisCmds = hashToRedisCmds(ki.key, val, batchSize)

		case "zset":
			var val []string
			if err = client.Do(cmd(&val, "ZRANGEBYSCORE", ki.key, "-inf", "+inf", "WITHSCORES")); err != nil {
				return err
			}
			redisCmds = zsetToRedisCmds(ki.key, val, batchSize)

		case "none":
			// Skip keys that don't exist (deleted between SCAN and TYPE)
			continue

		default:
			return fmt.Errorf("Key %s is of unrecognized type %s", ki.key, ki.keyType)
		}

		for _, redisCmd := range redisCmds {
			logger.Print(serializer(redisCmd))
		}

		if withTTL {
			var ttl int64
			if useRandomTTL {
				// Use random TTL instead of querying Redis
				ttl = generateRandomTTL(minRandomTTL, maxRandomTTL)
			} else {
				// Query TTL from Redis
				if err = client.Do(cmd(&ttl, "TTL", ki.key)); err != nil {
					return err
				}
			}
			if ttl > 0 {
				cmd := ttlToRedisCmd(ki.key, ttl)
				logger.Print(serializer(cmd))
			}
		}
	}

	return nil
}

func dumpKeysWorker(client radix.Client, keyBatches <-chan []string, skipFilters []string, withTTL bool, minRandomTTL int, maxRandomTTL int, batchSize int, logger *log.Logger, serializer Serializer, errors chan<- error, done chan<- bool) {
	for keyBatch := range keyBatches {
		if err := dumpKeys(client, radix.Cmd, keyBatch, skipFilters, withTTL, minRandomTTL, maxRandomTTL, batchSize, logger, serializer); err != nil {
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

		dbs = append(dbs, uint8(dbIndex))
	}

	return dbs, nil
}

func getDBIndexes(client *radix.Pool) ([]uint8, error) {
	var keyspaceInfo string
	if err := client.Do(radix.Cmd(&keyspaceInfo, "INFO", "keyspace")); err != nil {
		return nil, err
	}

	return parseKeyspaceInfo(keyspaceInfo)
}

func scanKeys(client radix.Client, cmd radixCmder, db uint8, keyBatchSize int, filter string, keyBatches chan<- []string, progressNotifications chan<- ProgressNotification) error {
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

func scanKeysLegacy(client radix.Client, cmd radixCmder, db uint8, keyBatchSize int, filter string, keyBatches chan<- []string, progressNotifications chan<- ProgressNotification) error {
	var err error
	var keys []string
	if err = client.Do(cmd(&keys, "KEYS", filter)); err != nil {
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
func RedisURL(redisHost string, redisPort string) string {
	return fmt.Sprintf("redis://%s:%s", redisHost, redisPort)
}

func redisDialOpts(redisUsername string, redisPassword string, tlsHandler *TlsHandler, db *uint8) ([]radix.DialOpt, error) {
	dialOpts := []radix.DialOpt{
		radix.DialTimeout(5 * time.Minute),
	}
	if redisPassword != "" {
		if redisUsername != "" {
			dialOpts = append(dialOpts, radix.DialAuthUser(redisUsername, redisPassword))
		} else {
			dialOpts = append(dialOpts, radix.DialAuthPass(redisPassword))
		}
	}
	if tlsHandler != nil {
		tlsCfg, err := tlsConfig(tlsHandler)
		if err != nil {
			return nil, err
		}
		dialOpts = append(dialOpts, radix.DialUseTLS(tlsCfg))
	}

	if db != nil {
		dialOpts = append(dialOpts, radix.DialSelectDB(int(*db)))
	}

	return dialOpts, nil
}

func dumpDB(client radix.Client, db *uint8, filter string, skipFilters []string, nWorkers int, withTTL bool, minRandomTTL int, maxRandomTTL int, batchSize int, noscan bool, logger *log.Logger, serializer Serializer, progress chan<- ProgressNotification) error {
	keyGenerator := scanKeys
	if noscan {
		keyGenerator = scanKeysLegacy
	}

	logger.Print(serializer([]string{"SELECT", fmt.Sprint(*db)}))

	errors := make(chan error)
	nErrors := 0
	go func() {
		for err := range errors {
			fmt.Fprintln(os.Stderr, "Error: "+err.Error())
			nErrors++
		}
	}()

	done := make(chan bool)
	keyBatches := make(chan []string)
	for i := 0; i < nWorkers; i++ {
		go dumpKeysWorker(client, keyBatches, skipFilters, withTTL, minRandomTTL, maxRandomTTL, batchSize, logger, serializer, errors, done)
	}

	keyGenerator(client, radix.Cmd, *db, 100, filter, keyBatches, progress)
	close(keyBatches)

	for i := 0; i < nWorkers; i++ {
		<-done
	}

	return nil
}

type Host struct {
	Host       string
	Port       int
	Username   string
	Password   string
	TlsHandler *TlsHandler
}

// DumpServer dumps all Keys from the redis server given by redisURL,
// to the Logger logger. Progress notification informations
// are regularly sent to the channel progressNotifications
func DumpServer(s Host, db *uint8, filter string, skipFilters []string, nWorkers int, withTTL bool, minRandomTTL int, maxRandomTTL int, batchSize int, noscan bool, logger *log.Logger, serializer func([]string) string, progress chan<- ProgressNotification) error {
	redisURL := RedisURL(s.Host, fmt.Sprint(s.Port))
	getConnFunc := func(db *uint8) func(network, addr string) (radix.Conn, error) {
		return func(network, addr string) (radix.Conn, error) {
			dialOpts, err := redisDialOpts(s.Username, s.Password, s.TlsHandler, db)
			if err != nil {
				return nil, err
			}

			return radix.Dial(network, addr, dialOpts...)
		}
	}

	dbs := []uint8{}
	if db != AllDBs {
		dbs = []uint8{*db}
	} else {
		client, err := radix.NewPool("tcp", redisURL, nWorkers, radix.PoolConnFunc(getConnFunc(nil)))
		if err != nil {
			return err
		}

		dbs, err = getDBIndexes(client)
		if err != nil {
			return err
		}
		client.Close()
	}

	for _, db := range dbs {
		client, err := radix.NewPool("tcp", redisURL, nWorkers, radix.PoolConnFunc(getConnFunc(&db)))
		if err != nil {
			return err
		}
		defer client.Close()

		if err = dumpDB(client, &db, filter, skipFilters, nWorkers, withTTL, minRandomTTL, maxRandomTTL, batchSize, noscan, logger, serializer, progress); err != nil {
			return err
		}
	}

	return nil
}
