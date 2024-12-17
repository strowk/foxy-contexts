package foxytest

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type diff struct {
	items []diffItem
}

func assertMatch(t TestRunner, expectation *yaml.Node, actual any) bool {
	diffResult := match(expectation, actual)
	if len(diffResult.items) > 0 {
		t.Errorf("\n%s", diffResult.String())
		return false
	}
	return true
}

// String returns a string representation of the diff
//
// The string representation is a convenient human-readable way
// to see what is different between the expectation and the actual
// values, that is using JSON as an assumed source of data and
// partly prints output in JSON format.
// The structure of diff is printed as JSON, but the diff values
// are just text explaining what was expected and what was found.
// For example if diff was for path ["a", "b"] and the message was
// "key not found", the output would be:
//
//	{
//	  "a": {
//	    "b":
//	     ^ key not found
//
// This also includes grouping, as in case if there are multiple
// diffs for the same path, they are grouped together, for example
// for path ["a", "b"] and messages "key not found" and "value mismatch" at path ["a", "b", "c"],
// the output would be:
//
//	{
//	  "a": {
//	    "b": {
//	     ^ key not found
//	       "c":
//	        ^ value mismatch
func (d *diff) String() string {
	result := ""
	// tree :=  d.buildTree()

	sort.Slice(d.items, func(i, j int) bool {
		// lexicographically sort the paths would allow to rely on path order
		return strings.Compare(strings.Join(d.items[i].path, ""), strings.Join(d.items[j].path, "")) < 0
	})

	previousPath := []string{}

	for _, item := range d.items {
		path := item.path
		var subpath []string
		message := item.message

		// calculate the subpath that is things that are unique to this path
		for i := 0; i < len(path); i++ {
			if i >= len(previousPath) || path[i] != previousPath[i] {
				subpath = append(subpath, path[i:]...)
				break
			} else {
				if i < len(previousPath) {
					result += "  "
				}
			}
		}

		// build the path in the tree
		for i, key := range subpath {
			if i == 0 && len(subpath) == 0 {
				result += "{\n"
			}
			indent := strings.Repeat("  ", i+1)
			result += fmt.Sprintf("%s\"%s\": {\n", indent, key)
		}

		// add the message
		indent := strings.Repeat("  ", len(path))
		result += fmt.Sprintf("%s^ %s\n", indent, message)

		previousPath = path
	}

	return result
}

type diffItem struct {
	path    []string
	message string
}

func pathCopy(path []string) []string {
	newPath := make([]string, len(path))
	copy(newPath, path)
	return newPath
}

func match(expectation *yaml.Node, actual any) *diff {
	diffResult := &diff{}
	diffResult.checkMatch(expectation, actual)
	return diffResult
}

func (d *diff) checkMatch(
	expectation *yaml.Node,
	actual any,
	currentPath ...string,
) {

	if expectation.Kind == yaml.DocumentNode {
		if len(expectation.Content) == 0 {
			return
		}

		d.checkMatch(expectation.Content[0], actual, currentPath...)
		return
	}

	// check if the expectation is a sequence
	if expectation.Kind == yaml.SequenceNode {

		// check if the actual is a slice
		actualSlice, ok := actual.([]interface{})
		if !ok {
			d.items = append(d.items, diffItem{
				path:    pathCopy(currentPath),
				message: "actual is not a sequence",
			})
			return
		}

		// check if the length of the sequence is the same
		if len(expectation.Content) != len(actualSlice) {
			d.items = append(d.items, diffItem{
				path:    pathCopy(currentPath),
				message: fmt.Sprintf("sequence length mismatch, expected: %d, got: %d", len(expectation.Content), len(actualSlice)),
			})
			return
		}

		// iterate over the expectation
		for i := 0; i < len(expectation.Content); i++ {
			path := append(currentPath, fmt.Sprintf("%d", i))
			d.checkMatch(expectation.Content[i], actualSlice[i], path...)
		}
		return
	}

	// check if the expectation is a map
	if expectation.Kind == yaml.MappingNode {

		// check if the actual is a map
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			d.items = append(d.items, diffItem{
				path:    pathCopy(currentPath),
				message: "actual is not a map",
			})
			return
		}

		// iterate over the expectation
		for i := 0; i < len(expectation.Content); i += 2 {
			key := expectation.Content[i].Value
			value := expectation.Content[i+1]

			path := append(currentPath, key)

			var actualValue interface{}
			var valueExists bool

			// check if the key exists in the actual map
			if actualValue, valueExists = actualMap[key]; !valueExists {
				d.items = append(d.items, diffItem{
					path:    pathCopy(path),
					message: "key not found",
				})
				continue
			}

			d.checkMatch(value, actualValue, path...)
		}
	}

	// check if the value is a scalar
	if expectation.Kind == yaml.ScalarNode {
		var decodedValue interface{}
		err := expectation.Decode(&decodedValue)
		if err != nil {
			d.items = append(d.items, diffItem{
				path:    pathCopy(currentPath),
				message: fmt.Sprintf("failed to decode expectation value '%s' : %v", expectation.Value, err),
			})
			return
		}

		if expectation.Tag == "!!re" {
			// this is regex

			// we should also check if the actual value is a string
			actualStringValue, ok := actual.(string)
			if !ok {
				d.items = append(d.items, diffItem{
					path:    pathCopy(currentPath),
					message: "actual value is not a string, but matched against a regex",
				})
				return
			}

			expectedStringValue, ok := decodedValue.(string)
			if !ok {
				d.items = append(d.items, diffItem{
					path:    currentPath,
					message: "expected value is not a string, but tagged as regex",
				})
				return
			}

			// check if the actual value matches the regex
			matches := matchRegex(expectedStringValue, actualStringValue)
			if !matches {
				d.items = append(d.items, diffItem{
					path:    pathCopy(currentPath),
					message: "value mismatch, actual value does not match the regex: " + expectedStringValue,
				})
			}
			return
		}

		if expectation.Tag == "!!ere" {
			// this is embedded regex

			// we should also check if the actual value is a string
			actualStringValue, ok := actual.(string)
			if !ok {
				d.items = append(d.items, diffItem{
					path:    pathCopy(currentPath),
					message: "actual value is not a string, but matched against an embedded regex",
				})
				return
			}

			expectedStringValue, ok := decodedValue.(string)
			if !ok {
				d.items = append(d.items, diffItem{
					path:    pathCopy(currentPath),
					message: "expected value is not a string, but tagged as embedded regex",
				})
				return
			}

			// check if the actual value matches the regex
			matches, _ := matchEmbeddedRegex(expectedStringValue, actualStringValue)
			if !matches {
				d.items = append(d.items, diffItem{
					path: pathCopy(currentPath),
					message: fmt.Sprintf(`value does not match the embedded regex: 
expected to match: "%s",
          but got: "%s"`, expectedStringValue, actualStringValue),
				})
				return
			}
			return
		}

		// when values are numbers, they might be different types
		// so the easies comparison of scalars is by their string representation

		if fmt.Sprintf("%v", decodedValue) != fmt.Sprintf("%v", actual) {
			d.items = append(d.items, diffItem{
				path: pathCopy(currentPath),
				message: fmt.Sprintf("value mismatch, \nexpected %T: '%v',\n     got %T: '%v'",
					decodedValue,
					decodedValue,
					actual,
					actual,
				),
			})
		}
		return
	}
}

const regexSymbolsToEscape = `[]{}()^$.|*+?`

func matchEmbeddedRegex(expected, actual string) (bool, string) {
	//we should locate everything in it that is between / and / and check if it is a valid regex
	// if it is not, we should fail the test, if it is, we should be matching it
	// against the value at the same place in the actual string
	// all slashes escaped by backslash should be unescaped, slashes
	// inside of the regex can be escaped by backslash

	// the easiest way is to build a regex that would work the same way as the bunch of embedded regexes
	var joinedRegex string
	for i := 0; i < len(expected); i++ {
		if expected[i] == '/' {
			// we should find the closing slash
			closingSlash := strings.Index(expected[i+1:], "/")
			if closingSlash == -1 {
				// there is no closing slash, this is an error
				return false, "failed to parse embedded regex, no closing slash"
			}
			// we should add the regex to the joined regex
			joinedRegex += expected[i+1 : i+1+closingSlash]
			// we should skip the regex
			i += closingSlash + 1
		} else {
			if expected[i] == '\\' {
				// we should unescape the slash
				if i+1 < len(expected) {
					next := expected[i+1]
					if next == '/' {
						joinedRegex += "/"
						i++
					} else {
						joinedRegex += "\\"
					}
				}
			} else {
				// as this part of the string is not a regex, we should escape all the regex symbols
				if strings.Contains(regexSymbolsToEscape, string(expected[i])) {
					joinedRegex += "\\" + string(expected[i])
				} else {
					joinedRegex += string(expected[i])
				}
			}
		}
	}

	// we should now match the actual string against the regex
	return matchRegex(joinedRegex, actual), joinedRegex
}

func matchRegex(regex, actual string) bool {
	// we should build the regex
	re, err := regexp.Compile(regex)
	if err != nil {
		// the regex is not valid
		return false
	}

	// we should match the actual string against the regex
	return re.MatchString(actual)
}
