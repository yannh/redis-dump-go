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
		s := genRedisProto(test.command)
		if s != test.expected {
			t.Errorf("Failed serializing command to redis protocl: expected %s, got %s", test.expected, s)
		}
	}

}
