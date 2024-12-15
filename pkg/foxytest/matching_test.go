package foxytest

import (
	"encoding/json"
	reflect "reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMatching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		expectation  string
		actual       string
		expectedDiff *diff
	}{

		{
			name: "simple map",
			expectation: `
a: 1
b: 2
`,
			actual:       `{"a": 1, "b": 2}`,
			expectedDiff: &diff{},
		},
		{
			name: "missing key",
			expectation: `
a: 1
b: 2
`,
			actual: `{"a": 1}`,
			expectedDiff: &diff{
				items: []diffItem{
					{
						path:    []string{"b"},
						message: "key not found",
					},
				},
			},
		},
		{
			name: "extra key",
			expectation: `
a: 1
`,
			actual:       `{"a": 1, "b": 2}`,
			expectedDiff: &diff{}, // this is not implemented yet
			// TODO: implement extra key diff
		},
		{
			name: "nested map",
			expectation: `
a:
  b: 1
  c: 2
`,
			actual:       `{"a": {"b": 1, "c": 2}}`,
			expectedDiff: &diff{},
		},
		{
			name: "nested map missing key",
			expectation: `
a:
  b: 1
  c: 2
`,
			actual: `{"a": {"b": 1}}`,
			expectedDiff: &diff{
				items: []diffItem{
					{
						path:    []string{"a", "c"},
						message: "key not found",
					},
				},
			},
		},
		{
			name: "nested map extra key",
			expectation: `
a:
  b: 1
`,
			actual:       `{"a": {"b": 1, "c": 2}}`,
			expectedDiff: &diff{}, // this is not implemented yet
			// TODO: implement extra key diff
		},
		{
			name: "sequence of strings",
			expectation: `
a:
  - b
  - c
`,
			actual:       `{"a": ["b", "c"]}`,
			expectedDiff: &diff{},
		},
		{
			name: "sequence of strings missing element",
			expectation: `
a:
  - b
  - c
`,
			actual: `{"a": ["b"]}`,
			expectedDiff: &diff{
				items: []diffItem{
					{
						path:    []string{"a"},
						message: "sequence length mismatch, expected: 2, got: 1",
					},
				},
			},
		},
		{
			name: "sequence of strings extra element",
			expectation: `
a:
  - b
`,
			actual: `{"a": ["b", "c"]}`,
			expectedDiff: &diff{
				items: []diffItem{
					{
						path:    []string{"a"},
						message: "sequence length mismatch, expected: 1, got: 2",
					},
				},
			},
		},
		{
			name: "sequence of maps",
			expectation: `
a:
  - b: 1
    c: 2
  - b: 3
    c: 4
`,
			actual:       `{"a": [{"b": 1, "c": 2}, {"b": 3, "c": 4}]}`,
			expectedDiff: &diff{},
		},
		{
			name: "sequence of maps missing nested element",
			expectation: `
a:
  - b: 1
    c: 2
  - b: 3
    c: 4
`,
			actual: `{"a": [{"b": 1, "c": 2}, {"b": 3}]}`,
			expectedDiff: &diff{
				items: []diffItem{
					{
						path:    []string{"a", "1", "c"},
						message: "key not found",
					},
				},
			},
		},
		{
			name: "three level nested map with wrong value",
			expectation: `
a:
  b:
    c: "hmm"
    d: 4
`,
			actual: `{"a": {"b": {"c": "oh oh", "d": 4}}}`,
			expectedDiff: &diff{
				items: []diffItem{
					{
						path:    []string{"a", "b", "c"},
						message: "value mismatch, \nexpected string: 'hmm',\n     got string: 'oh oh'",
					},
				},
			},
		},
		{
			name: "sequence of maps wrong nested element",
			expectation: `
a:
  b:
    - c: "hmm"
      d: 4
`,
			actual: `{"a": {"b": [{"c": "oh oh", "d": 4}]}}`,
			expectedDiff: &diff{
				items: []diffItem{
					{
						path:    []string{"a", "b", "0", "c"},
						message: "value mismatch, \nexpected string: 'hmm',\n     got string: 'oh oh'",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectation := &yaml.Node{}
			err := yaml.Unmarshal([]byte(tt.expectation), expectation)
			if err != nil {
				t.Fatal(err)
			}

			actual := map[string]interface{}{}
			err = json.Unmarshal([]byte(tt.actual), &actual)
			if err != nil {
				t.Fatal(err)
			}

			diffResult := match(expectation, actual)
			if !reflect.DeepEqual(diffResult, tt.expectedDiff) {
				t.Errorf("diff got = '\n%v', want '\n%v'", diffResult, tt.expectedDiff)
			}
		})
	}
}

func TestDiffPrint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		diff *diff
		want string
	}{
		{
			name: "simple diff",
			diff: &diff{
				items: []diffItem{
					{
						path:    []string{"a"},
						message: "key not found",
					},
				},
			},
			want: `
  "a": {
  ^ key not found`,
		},
		{
			name: "nested diff",
			diff: &diff{
				items: []diffItem{
					{
						path:    []string{"a", "b"},
						message: "key not found",
					},
					{
						path:    []string{"a", "c"},
						message: "key not found",
					},
				},
			},
			want: `
"a": {
    "b": {
    ^ key not found
    "c": {
    ^ key not found`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diff.String(); strings.TrimSpace(got) != strings.TrimSpace(tt.want) {
				t.Errorf("got = \n'%v', want \n'%v'", strings.TrimSpace(got), strings.TrimSpace(tt.want))
			}
		})
	}
}
