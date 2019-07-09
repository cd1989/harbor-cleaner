package clean

import "testing"

func TestRetain(t *testing.T) {
	cases := []struct {
		tag            string
		retainPatterns []string
		expected       bool
	}{
		{
			"v1.0",
			[]string{"v1.0"},
			true,
		},
		{
			"v1.0",
			[]string{"v1.?"},
			true,
		},
		{
			"v1.0",
			[]string{"v1.1", "v1.*"},
			true,
		},
		{
			"v1.0",
			[]string{"v*"},
			true,
		},
		{
			"v1.0",
			[]string{"v?.0"},
			true,
		},
		{
			"v1.0",
			[]string{},
			false,
		},
		{
			"v1.0",
			[]string{"v1.1"},
			false,
		},
		{
			"v2.0",
			[]string{"v1.*"},
			false,
		},
	}

	for _, c := range cases {
		actual := retain(c.retainPatterns, c.tag)
		if actual != c.expected {
			t.Errorf("Retain '%s' against %v expected to be %v, but got %v", c.tag, c.retainPatterns, c.expected, actual)
		}
	}
}
