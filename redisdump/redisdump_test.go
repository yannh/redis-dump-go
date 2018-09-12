package redisdump

import "testing"

func testEq(a, b []string) bool {

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
		if !testEq(res, test.expected) {
			t.Errorf("Failed generating redis command from string for: %s %s", test.key, test.value)
		}
	}

}

func TestHashToRedisCmd(t *testing.T) {
	type testCase struct {
		key      string
		value    map[string]string
		expected []string
	}

	testCases := []testCase{
		{key: "Paris", value: map[string]string{"country": "France", "weather": "sunny"}, expected: []string{"HSET", "Paris", "country", "France", "weather", "sunny"}},
	}

	for _, test := range testCases {
		res := hashToRedisCmd(test.key, test.value)
		if !testEq(res, test.expected) {
			t.Errorf("Failed generating redis command from Hash for: %s %s", test.key, test.value)
		}
	}
}

func TestZsetToRedisCmd(t *testing.T) {
	type testCase struct {
		key      string
		value    []string
		expected []string
	}

	testCases := []testCase{
		{key: "todo", value: []string{"task1", "1", "task2", "2", "task3", "3"}, expected: []string{"ZADD", "todo", "1", "task1", "2", "task2", "3", "task3"}},
	}

	for _, test := range testCases {
		res := zsetToRedisCmd(test.key, test.value)
		if !testEq(res, test.expected) {
			t.Errorf("Failed generating redis command from Hash for: %s %s, got %v", test.key, test.value, res)
		}
	}

}

func TestGenRedisProto(t *testing.T) {
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
