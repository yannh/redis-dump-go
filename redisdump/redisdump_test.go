package redisdump

import (
	"testing"
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
			for j := 2; j < len(res[i]); j+=2 {
				found := false
				for k := 0; k<len(test.expected); k++ {
					for l := 2; l < len(test.expected[k]); l+=2 {
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
		}

		for i := 0; i<len(testCase.expected); i++ {
			if len(testCase.expected[i]) != len(res[i]) {
				t.Errorf("Failed generating redis command from SET for %s %s %d: got %s", testCase.key, testCase.value, testCase.cmdMaxLen, res)
			}
			for j := 0; j<len(testCase.expected[i]); j++ {
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

	for _, test := range testCases {
		res := zsetToRedisCmds(test.key, test.value, 1)
		for i := 0; i < len(res); i++ {
			for j := 2; j < len(res[i]); j+=2 {
				found := false
				for k := 0; k<len(test.expected); k++ {
					for l := 2; l < len(test.expected[k]); l+=2 {
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
