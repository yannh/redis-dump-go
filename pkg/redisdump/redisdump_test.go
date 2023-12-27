package redisdump

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"testing"

	"github.com/mediocregopher/radix/v3"
)

func testEqString(a, b []string) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func testEqUint8(a, b []uint8) bool {
	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestStringToRedisCmd(t *testing.T) {
	type testCase struct {
		key, value string
		expected   []string
	}

	testCases := []testCase{
		{key: "city", value: "Paris", expected: []string{"SET", "city", "Paris"}},
		{key: "fullname", value: "Jean-Paul Sartre", expected: []string{"SET", "fullname", "Jean-Paul Sartre"}},
		{key: "unicode", value: "😈", expected: []string{"SET", "unicode", "😈"}},
	}

	for _, test := range testCases {
		res := stringToRedisCmd(test.key, test.value)
		if !testEqString(res, test.expected) {
			t.Errorf("Failed generating redis command from string for: %s %s", test.key, test.value)
		}
	}
}

func TestHashToRedisCmds(t *testing.T) {
	type testCase struct {
		key       string
		value     map[string]string
		cmdMaxLen int
		expected  [][]string
	}

	testCases := []testCase{
		{key: "Paris", value: map[string]string{"country": "France", "weather": "sunny", "poi": "Tour Eiffel"}, cmdMaxLen: 1, expected: [][]string{{"HSET", "Paris", "country", "France"}, {"HSET", "Paris", "weather", "sunny"}, {"HSET", "Paris", "poi", "Tour Eiffel"}}},
		{key: "Paris", value: map[string]string{"country": "France", "weather": "sunny", "poi": "Tour Eiffel"}, cmdMaxLen: 2, expected: [][]string{{"HSET", "Paris", "country", "France", "weather", "sunny"}, {"HSET", "Paris", "poi", "Tour Eiffel"}}},
		{key: "Paris", value: map[string]string{"country": "France", "weather": "sunny", "poi": "Tour Eiffel"}, cmdMaxLen: 3, expected: [][]string{{"HSET", "Paris", "country", "France", "weather", "sunny", "poi", "Tour Eiffel"}}},
		{key: "Paris", value: map[string]string{"country": "France", "weather": "sunny", "poi": "Tour Eiffel"}, cmdMaxLen: 4, expected: [][]string{{"HSET", "Paris", "country", "France", "weather", "sunny", "poi", "Tour Eiffel"}}},
	}

	for _, test := range testCases {
		res := hashToRedisCmds(test.key, test.value, test.cmdMaxLen)
		for i := 0; i < len(res); i++ {
			for j := 2; j < len(res[i]); j += 2 {
				found := false
				for k := 0; k < len(test.expected); k++ {
					for l := 2; l < len(test.expected[k]); l += 2 {
						if res[i][j] == test.expected[k][l] && res[i][j+1] == test.expected[k][l+1] {
							found = true
						}
					}
				}

				if found == false {
					t.Errorf("Failed generating redis command from Hash for: %s %s, got %s", test.key, test.value, res)
				}
			}
		}
	}
}

func TestSetToRedisCmds(t *testing.T) {
	type testCase struct {
		key       string
		value     []string
		cmdMaxLen int
		expected  [][]string
	}

	testCases := []testCase{
		{key: "myset", value: []string{"1", "2", "3"}, cmdMaxLen: 1, expected: [][]string{{"SADD", "myset", "1"}, {"SADD", "myset", "2"}, {"SADD", "myset", "3"}}},
		{key: "myset", value: []string{"1", "2", "3"}, cmdMaxLen: 2, expected: [][]string{{"SADD", "myset", "1", "2"}, {"SADD", "myset", "3"}}},
		{key: "myset", value: []string{"1", "2", "3"}, cmdMaxLen: 3, expected: [][]string{{"SADD", "myset", "1", "2", "3"}}},
		{key: "myset", value: []string{"1", "2", "3"}, cmdMaxLen: 4, expected: [][]string{{"SADD", "myset", "1", "2", "3"}}},
	}

	for _, testCase := range testCases {
		res := setToRedisCmds(testCase.key, testCase.value, testCase.cmdMaxLen)
		if len(testCase.expected) != len(res) {
			t.Errorf("Failed generating redis command from SET for %s %s %d: got %s", testCase.key, testCase.value, testCase.cmdMaxLen, res)
			continue
		}

		for i := 0; i < len(testCase.expected); i++ {
			if len(testCase.expected[i]) != len(res[i]) {
				t.Errorf("Failed generating redis command from SET for %s %s %d: got %s", testCase.key, testCase.value, testCase.cmdMaxLen, res)
				continue
			}
			for j := 0; j < len(testCase.expected[i]); j++ {
				if res[i][j] != testCase.expected[i][j] {
					t.Errorf("Failed generating redis command from SET for %s %s %d: got %s", testCase.key, testCase.value, testCase.cmdMaxLen, res)
				}
			}
		}
	}
}

func TestZsetToRedisCmds(t *testing.T) {
	type testCase struct {
		key       string
		value     []string
		cmdMaxLen int
		expected  [][]string
	}

	testCases := []testCase{
		{key: "todo", value: []string{"task1", "1", "task2", "2", "task3", "3"}, cmdMaxLen: 1, expected: [][]string{{"ZADD", "todo", "1", "task1"}, {"ZADD", "todo", "2", "task2"}, {"ZADD", "todo", "3", "task3"}}},
		{key: "todo", value: []string{"task1", "1", "task2", "2", "task3", "3"}, cmdMaxLen: 2, expected: [][]string{{"ZADD", "todo", "1", "task1", "2", "task2"}, {"ZADD", "todo", "3", "task3"}}},
		{key: "todo", value: []string{"task1", "1", "task2", "2", "task3", "3"}, cmdMaxLen: 3, expected: [][]string{{"ZADD", "todo", "1", "task1", "2", "task2", "3", "task3"}}},
		{key: "todo", value: []string{"task1", "1", "task2", "2", "task3", "3"}, cmdMaxLen: 4, expected: [][]string{{"ZADD", "todo", "1", "task1", "2", "task2", "3", "task3"}}},
	}

	for _, testCase := range testCases {
		res := zsetToRedisCmds(testCase.key, testCase.value, testCase.cmdMaxLen)
		if len(testCase.expected) != len(res) {
			t.Errorf("Failed generating redis command from ZSET for %s %s %d: got %s", testCase.key, testCase.value, testCase.cmdMaxLen, res)
			continue
		}
		for i := 0; i < len(res); i++ {
			if len(testCase.expected[i]) != len(res[i]) {
				t.Errorf("Failed generating redis command from ZSET for %s %s %d: got %s", testCase.key, testCase.value, testCase.cmdMaxLen, res)
				continue
			}
			for j := 2; j < len(res[i]); j += 2 {
				found := false
				if res[i][j] == testCase.expected[i][j] && res[i][j+1] == testCase.expected[i][j+1] {
					found = true
				}

				if found == false {
					t.Errorf("Failed generating redis command from ZSet for: %s %s %d, got %s", testCase.key, testCase.value, testCase.cmdMaxLen, res)
				}
			}
		}
	}
}

func TestRESPSerializer(t *testing.T) {
	type testCase struct {
		command  []string
		expected string
	}

	testCases := []testCase{
		{command: []string{"SET", "key", "value"}, expected: "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"},
		{command: []string{"SET", "key1", "😈"}, expected: "*3\r\n$3\r\nSET\r\n$4\r\nkey1\r\n$4\r\n😈\r\n"},
	}

	for _, test := range testCases {
		s := RESPSerializer(test.command)
		if s != test.expected {
			t.Errorf("Failed serializing command to redis protocol: expected %s, got %s", test.expected, s)
		}
	}
}

func TestRedisCmdSerializer(t *testing.T) {
	type testCase struct {
		command  []string
		expected string
	}

	testCases := []testCase{
		{command: []string{"HELLO"}, expected: "HELLO"},
		{command: []string{"HGETALL", "key"}, expected: "HGETALL key"},
		{command: []string{"SET", "key name 1", "key value 1"}, expected: "SET \"key name 1\" \"key value 1\""},
		{command: []string{"SET", "key", ""}, expected: "SET key \"\""},
		{command: []string{"HSET", "key1", "key value 1"}, expected: "HSET key1 \"key value 1\""},
	}

	for _, test := range testCases {
		s := RedisCmdSerializer(test.command)
		if s != test.expected {
			t.Errorf("Failed serializing command to redis protocol: expected %s, got %s", test.expected, s)
		}
	}
}

func TestParseKeyspaceInfo(t *testing.T) {
	keyspaceInfo := `# Keyspace
	db0:keys=2,expires=1,avg_ttl=1009946407050
	db2:keys=1,expires=0,avg_ttl=0`

	dbIds, err := parseKeyspaceInfo(keyspaceInfo)
	if err != nil {
		t.Errorf("Failed parsing keyspaceInfo" + err.Error())
	}
	if !testEqUint8(dbIds, []uint8{0, 2}) {
		t.Errorf("Failed parsing keyspaceInfo: got %v", dbIds)
	}
}

func TestRedisDialOpts(t *testing.T) {
	for i, testCase := range []struct {
		redisUsername string
		redisPassword string
		tlsHandler    *TlsHandler
		db            uint8
		nDialOpts     int
		err           error
	}{
		{
			"",
			"",
			nil,
			1,
			2,
			nil,
		}, {
			"",
			"test",
			&TlsHandler{},
			1,
			4,
			nil,
		}, {
			"test",
			"test",
			&TlsHandler{},
			1,
			4,
			nil,
		},
	} {
		dOpts, err := redisDialOpts(testCase.redisUsername, testCase.redisPassword, testCase.tlsHandler, &testCase.db)
		if err != testCase.err {
			t.Errorf("expected error to be %+v, got %+v", testCase.err, err)
		}

		// DialOpts are functions and are pretty difficult to compare :(
		// "Functions are equal only if they are both nil"
		// Therefore we only compare that we are getting the right amount
		if len(dOpts) != testCase.nDialOpts {
			t.Errorf("test %d, expected %d dialOpts, got %d", i, testCase.nDialOpts, len(dOpts))
		}

	}
}

type mockRadixClient struct{}

func (m *mockRadixClient) Do(action radix.Action) error {
	return action.Run(nil)
}
func (m *mockRadixClient) Close() error {
	return nil
}

type mockRadixAction struct {
	rcv  interface{}
	cmd  string
	args []string
}

func (m *mockRadixAction) Keys() []string {
	return nil
}

func (m *mockRadixAction) Run(conn radix.Conn) error {
	if m.cmd == "TYPE" {
		key := m.args[0]
		// if the key name contains string, the object is of type string
		if strings.Contains(key, "string") {
			switch v := m.rcv.(type) {
			case *string:
				*v = "string"
			}
		}
		if strings.Contains(key, "list") {
			switch v := m.rcv.(type) {
			case *string:
				*v = "list"
			}
		}
		if strings.Contains(key, "zset") {
			switch v := m.rcv.(type) {
			case *string:
				*v = "zset"
			}
		}

		return nil
	}

	if m.cmd == "GET" {
		switch v := m.rcv.(type) {
		case *string:
			*v = "stringvalue"
		default:
			fmt.Printf("DEFAULT")
		}

		return nil
	}

	if m.cmd == "TTL" {
		switch v := m.rcv.(type) {
		case *int64:
			*v = 5
		}

		return nil
	}

	if m.cmd == "KEYS" {
		switch v := m.rcv.(type) {
		case *[]string:
			a := []string{"key1", "key2", "key3", "key4", "key5"}
			*v = a
		}

		return nil
	}

	if m.cmd == "LRANGE" {
		switch v := m.rcv.(type) {
		case *[]string:
			a := []string{"listkey1", "listval1", "listkey2", "listval2"}
			*v = a

		default:
			fmt.Printf("ERROR")
		}
		return nil
	}

	if m.cmd == "ZRANGEBYSCORE" {
		switch v := m.rcv.(type) {
		case *[]string:
			a := []string{"listkey1", "1", "listkey2", "2"}
			*v = a

		default:
			fmt.Printf("ERROR")
		}
		return nil
	}

	return nil
}

func (m *mockRadixAction) MarshalRESP(io.Writer) error {
	return nil
}

func (m *mockRadixAction) UnmarshalRESP(reader *bufio.Reader) error {
	return nil
}

func getMockRadixAction(rcv interface{}, cmd string, args ...string) radix.CmdAction {
	return &mockRadixAction{
		rcv:  rcv,
		cmd:  cmd,
		args: args,
	}
}

func TestDumpKeys(t *testing.T) {
	for i, testCase := range []struct {
		keys        []string
		withTTL     bool
		expectMatch string
	}{
		{
			[]string{"somestring"},
			false,
			"^SET somestring stringvalue\n$",
		},
		{
			[]string{"somestring", "somelist"},
			false,
			"^SET somestring stringvalue\nRPUSH somelist listkey1 listval1 listkey2 listval2\n$",
		},
		{
			[]string{"somestring"},
			true,
			"^SET somestring stringvalue\nEXPIREAT somestring [0-9]+\n$",
		},
		{
			[]string{"somezset"},
			false,
			"^ZADD somezset 1 listkey1 2 listkey2\n$",
		},
	} {
		var m mockRadixClient
		var b bytes.Buffer
		l := log.New(&b, "", 0)
		err := dumpKeys(&m, getMockRadixAction, testCase.keys, testCase.withTTL, 5, l, RedisCmdSerializer)
		if err != nil {
			t.Errorf("received error %+v", err)
		}
		match, _ := regexp.MatchString(testCase.expectMatch, b.String())
		if !match {
			t.Errorf("test %d: expected to match %s, got %s", i, testCase.expectMatch, b.String())
		}
	}
}

func TestScanKeysLegacy(t *testing.T) {
	for i, testCase := range []struct {
		n     int
		bSize int
		err   error
	}{
		{
			5,
			100,
			nil,
		},
		{
			5,
			4,
			nil,
		},
		{
			5,
			5,
			nil,
		},
	} {
		var m mockRadixClient
		keyBatches := make(chan []string)

		n := 0
		done := make(chan bool)
		go func() {
			for b := range keyBatches {
				n += len(b)
			}
			done <- true
		}()

		err := scanKeysLegacy(&m, getMockRadixAction, 0, 100, "*", keyBatches, nil)
		close(keyBatches)
		<-done
		if err != testCase.err {
			t.Errorf("test %d, expected err to be %s, got %s", i, testCase.err, err)
		}
		if n != testCase.n {
			t.Errorf("test %d, expected %d keys, got %d", i, testCase.n, n)
		}
	}
}
