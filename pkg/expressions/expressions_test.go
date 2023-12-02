package expressions

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		expr      string
		expected  [][]string
		shouldErr bool
	}{
		// with events
		{
			expr: `"lol" == event.data || event.name == "foo" || (user.source == 'test' || user.source == 'google') && date(user.next_visit) > now_plus("24h") && user.email == 'test@example.com'`,
			expected: [][]string{
				{"event", "data"},
				{"event", "name"},
				{"user", "email"},
				{"user", "next_visit"},
				{"user", "source"},
			},
		},
		// nested
		{
			expr: `event.data.action == 'opened'`,
			expected: [][]string{
				{"event", "data", "action"},
			},
		},
	}

	for _, test := range tests {
		expr, err := NewExpressionEvaluator(context.Background(), test.expr)
		require.NoError(t, err, test.expr)
		attrs := expr.UsedAttributes(context.Background()).FullPaths()
		require.Equal(t, err == nil, !test.shouldErr, "unexpected err result %s for '%s'", err, test.expr)
		require.Equal(t, len(test.expected), len(attrs), test.expr)
		require.ElementsMatch(t, test.expected, attrs, test.expr)
	}
}

func TestEvaluateExpression(t *testing.T) {
	tests := []struct {
		expr string
		data map[string]interface{}

		expected  any
		earliest  *time.Time
		shouldErr bool
		errMsg    string
	}{
		// Returning data
		{
			"event.data.name",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"name": "Tester McTestyFace",
					},
				},
			},
			"Tester McTestyFace",
			nil,
			false,
			"",
		},
		{
			"5 + 4",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"name": "Tester McTestyFace",
					},
				},
			},
			int64(9),
			nil,
			false,
			"",
		},
		{
			// concat str + number
			"event.data.name + '-' + event.data.number + '-' + event.data.bool + '-' + event.data.list + '-' + event.data.obj + '-' + event.data.missing",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"name":   "hi",
						"number": 9.1,
						"bool":   true,
						"list":   []any{3, "ok?", true},
						"obj": map[string]any{
							"name": "inngest",
						},
					},
				},
			},
			"hi-9.1-true-[3 ok? true]-map[name:inngest]-",
			nil,
			false,
			"",
		},
		{
			// missing attr
			"event.data.name + '-' + event.data.foo",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"name": "Tester McTestyFace",
					},
				},
			},
			"Tester McTestyFace-",
			nil,
			false,
			"",
		},
		{
			"uppercase(event.data.name) + ' is my name'",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"name": "Tester McTestyFace",
					},
				},
			},
			"TESTER MCTESTYFACE is my name",
			nil,
			false,
			"",
		},

		// Base64 decoding
		{
			`b64decode("aGkgdGhlcmUh") + " - b64 data"`,
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{},
				},
			},
			"hi there! - b64 data",
			nil,
			false,
			"",
		},
		{
			`b64decode("aGkgdGhlcmUh") == "hi there!"`,
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{},
				},
			},
			true,
			nil,
			false,
			"",
		},
		// Base64 decoding a JSON map and accesing object
		{
			`json_parse(b64decode(event.data.encoded)).nested.data`,
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"encoded": "eyJuZXN0ZWQiOnsiZGF0YSI6InllYSJ9fQ==",
						// this is the JSON-decoded base64 string above.
						"decoded": map[string]any{
							"nested": map[string]any{
								"data": "yea",
							},
						},
					},
				},
			},
			"yea",
			nil,
			false,
			"",
		},
		{
			`json_parse(b64decode(event.data.encoded)).nested.data == "yea"`,
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"encoded": "eyJuZXN0ZWQiOnsiZGF0YSI6InllYSJ9fQ==",
						// this is the JSON-decoded base64 string above.
						"decoded": map[string]any{
							"nested": map[string]any{
								"data": "yea",
							},
						},
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		// map equality
		{
			"event.data == event.data",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]any{
						"nope": nil,
						"inner": map[string]any{
							"eval": "ok",
						},
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.a == event.data.b",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]any{
						"nope": nil,
						"a": map[string]any{
							"eval": "ok",
							"num":  1.161,
						},
						"b": map[string]any{
							"eval": "ok",
							"num":  1.161,
						},
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.a == event.data.b",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]any{
						"nope": nil,
						"a": map[string]any{
							"eval": "ok",
							"num":  9,
						},
						"b": map[string]any{
							"eval": "ok",
							"num":  1.161,
						},
					},
				},
			},
			false,
			nil,
			false,
			"",
		},

		// Basic boolean expressions
		{
			"null > 0",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"nope": nil,
					},
				},
			},
			false,
			nil,
			false,
			"",
		},
		{
			"size(event.data.nope) > 0",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"nope": nil,
					},
				},
			},
			false,
			nil,
			false,
			"",
		},
		{
			"event.data.action > 0",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"action": nil,
					},
				},
			},
			false,
			nil,
			false,
			"",
		},
		{
			"event.data.float <= null && event.data.int < null",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"float": 3.141,
						"int":   8,
					},
				},
			},
			false,
			nil,
			false,
			"",
		},
		// For more information on available macros and overloads, look at
		// cel-go/common/operators/operators.go and
		// cel-go/common/overloads/overloads.go.
		// Type coercion
		{
			// comparing against unknowns / null
			"event.data.int > event.data.unknown",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"int": 2,
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.something == null",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"something": "hi",
					},
				},
			},
			false,
			nil,
			false,
			"",
		},
		{
			"event.data.float > null && event.data.int > null",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"float": 3.141,
						"int":   8,
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.float <= null && event.data.int < null",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"float": 3.141,
						"int":   8,
					},
				},
			},
			false,
			nil,
			false,
			"",
		},
		{
			"event.data.int > 3.141 && event.data.float <= 2",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"int":   8,
						"float": 1.111,
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		// nulls
		{
			"event.data.something == null",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"something": nil,
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.another == null",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{},
				},
			},
			true,
			nil,
			false,
			"",
		},
		// basics
		{
			expr: `steps.first.foo == 'test' && steps.second.bar != true && steps.second.baz != true`,
			data: map[string]interface{}{
				"steps": map[string]map[string]interface{}{
					"first": {
						"foo": "test",
					},
					"second": {
						"foo": "test",
					},
				},
			},
			expected:  true,
			earliest:  nil,
			shouldErr: false,
		},
		{
			expr: `actions.first.foo != 'test' && (actions.first.bar == true || actions.second.baz == true)`,
			data: map[string]interface{}{
				"actions": map[string]map[string]interface{}{
					"first": {
						"foo": "test",
					},
				},
			},
			expected:  false,
			earliest:  nil,
			shouldErr: false,
		},

		{
			"event.data.foo != '' && 'item' in event.data.issue.fields.tags",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{},
				},
			},
			false,
			nil,
			false,
			"",
		},
		{
			"event.data.foo != ''",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"foo": "hi",
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event['data'].foo != '' && event['data'].bar != true",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"foo": "hi",
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		// unknowns
		{
			"event.data.something == null",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"something": nil,
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.NOPE == null",
			map[string]interface{}{
				"event": event.Event{},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.name == action.first.result.name",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{"name": "no"},
				},
			},
			false,
			nil,
			false,
			"",
		},
		// deeply nested
		{
			"event.data.issue.field.another == 'another'",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"issue": nil,
					},
				},
			},
			false,
			nil,
			false,
			"",
		},
		{
			"event.data.issue.field.another == 'another'",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"issue": nil,
					},
				},
			},
			false,
			nil,
			false,
			"",
		},
		// deeply nested, testing non-existent fields.
		{
			"event.data.test == true && event.data.issue.field.another != 'another'",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"test":  true,
						"issue": nil,
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		// includes
		{
			expr:      `user.nonexistent.exists(x, x == 'a')`,
			expected:  false,
			earliest:  nil,
			shouldErr: false,
		},
		// steps
		{
			expr: `steps.first.foo == 'test'`,
			data: map[string]interface{}{
				"steps": map[string]interface{}{
					"first": map[string]interface{}{
						"foo": "test",
					},
				},
			},
			expected:  true,
			earliest:  nil,
			shouldErr: false,
		},
		// action
		{
			expr: `actions.first.foo == 'test' && actions.first.bar != true`,
			data: map[string]interface{}{
				"actions": map[string]map[string]interface{}{
					"first": {
						"foo": "test",
					},
				},
			},
			expected:  true,
			earliest:  nil,
			shouldErr: false,
		},
		{
			expr: `actions.first.foo == 'test' && actions.first.bar != true`,
			data: map[string]interface{}{
				"actions": map[string]map[string]interface{}{
					"first": {
						"foo": "test",
					},
				},
			},
			expected:  true,
			earliest:  nil,
			shouldErr: false,
		},
		// matches
		{
			`event.data.title.matches('^[a-z\\d]{0,10}$')`,
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"title": "nope",
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			`event.data.title.matches('\\w')`,
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"title": "nope",
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		// contains
		{
			"event.data.some.unknown.contains(event.data.title)",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"title": "nope",
					},
				},
			},
			false,
			nil,
			false,
			"",
		},
		{
			"'hello'.contains('llo')",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"action": "opened",
					},
				},
				"response": map[string]interface{}{"status": "200"},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"'hello'.contains('what')",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"action": "opened",
					},
				},
				"response": map[string]interface{}{"status": "200"},
			},
			false,
			nil,
			false,
			"",
		},
		// size text
		{
			"size(event.data.action) > 0",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"action": nil,
					},
				},
			},
			false,
			nil,
			false,
			"",
		},
		{
			"size(event.data.action) > 0",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"action": "opened",
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"size(event.data.action) == 6",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"action": "opened",
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		// inputs
		{
			// A complex input
			"event.data.action == 'opened'",
			map[string]interface{}{
				"event": event.Event{
					Data: map[string]interface{}{
						"action": "opened",
					},
				},
				"response": map[string]interface{}{"status": "200"},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"int(response.status) == 200",
			map[string]interface{}{
				"response": map[string]interface{}{"status": "200"},
			},
			true,
			nil,
			false,
			"",
		},
		// upper/lower
		{
			"uppercase(event.priority) == 'P1'",
			map[string]interface{}{
				"event": map[string]interface{}{
					"priority": "p1",
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"lowercase(event.priority) == 'urgent'",
			map[string]interface{}{
				"event": map[string]interface{}{
					"priority": "URGENT",
				},
			},
			true,
			nil,
			false,
			"",
		},
		// multiline
		{
			`
											int(response.status) >= 400 &&
											response.error == true
										`,
			map[string]interface{}{
				"response": map[string]interface{}{"status": "400", "error": true},
			},
			true,
			nil,
			false,
			"",
		},
		// Array funcs
		{
			"size(event.data.tags) == 0",
			map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"size(event.data.tags) == 1",
			map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{},
				},
			},
			false,
			nil,
			false,
			"",
		},
		{
			"size(event.data.tags) == 0",
			map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"tags": []string{},
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"size(event.data.tags) > 2",
			map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"tags": []string{"a", "b", "c"},
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.tags.exists(x, x == 'a')",
			map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"tags": []string{"a", "b", "c"},
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.tags.exists(x, x == 'd' || x == 'a')",
			map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"tags": []string{"a", "b", "c"},
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.tags.exists(x, x == 'd')",
			map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"tags": []string{"a", "b", "c"},
					},
				},
			},
			false,
			nil,
			false,
			"",
		},
		{
			"event.data.tags.exists(x, x == 'd') == false",
			map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"tags": []string{"a", "b", "c"},
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		// in
		{
			"event.data.issue in ['Bug', 'Issue', 'Epic'] == true",
			map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"issue": "Bug",
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.issue in ['LOL', 'Issue', 'Epic'] == false",
			map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"issue": "Bug",
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"event.data.nonexistent in ['LOL', 'Issue', 'Epic'] == false",
			map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"issue": "Bug",
					},
				},
			},
			true,
			nil,
			false,
			"",
		},
		// Struct + cel stdlib expressions
		{
			"[1, 2, 3].all(x, x > 0) && response.status == '200'",
			map[string]interface{}{
				"response": map[string]interface{}{"status": "200"},
			},
			true,
			nil,
			false,
			"",
		},
		{
			"response.status == 200",
			map[string]interface{}{
				"event":    map[string]interface{}{"foo": "bar"},
				"response": map[string]interface{}{"status": "200"},
			},
			false,
			nil,
			false,
			// no error, but it doesn't match.
			"",
		},
		// Event data time manipulation
		{
			"date(event.from) + duration('6h35m10s') == date('2020-01-01T18:35:10')",
			map[string]interface{}{
				"event": map[string]any{
					"from": "2020-01-01T12:00:00",
				},
			},
			true,
			nil,
			false,
			"",
		},
	}

	for n, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			actual, earliest, err := Evaluate(context.Background(), test.expr, test.data)

			require.Equal(t, err == nil, !test.shouldErr, "unexpected err result '%v' for '%s'", err, test.expr)
			require.Equal(t, test.expected, actual, "unexpected match result (%t): for (%d) '%s'", actual, n, test.expr)

			if test.shouldErr && err != nil {
				require.True(t, strings.Contains(err.Error(), test.errMsg), "Error should contain %s, got %s", test.errMsg, err.Error())
			}

			if test.earliest != nil {
				require.NotNil(t, earliest, test.expr)
				require.WithinDuration(
					t,
					*test.earliest,
					*earliest,
					time.Second,
					"invalid earliest time.  expected '%s', got '%s' for '%s'",
					test.earliest.Format(time.RFC3339),
					earliest.Format(time.RFC3339),
					test.expr,
				)
			} else {
				require.Nil(t, earliest, "expected nil earliest date for '%s'", test.expr)
			}
		})

	}
}

func TestFilteredAttributes(t *testing.T) {
	tests := []struct {
		expr     string
		in       *Data
		expected *Data
	}{
		{
			// one present, one mising
			expr: `event.data.name == "hi" && user.favourites == ['a']`,
			in: NewData(map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"name":  "hi",
						"email": "some",
					},
					"user": map[string]interface{}{
						"external_id": 1,
						"email":       "test@egypt.lol",
					},
				},
			}),
			expected: NewData(map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"name": "hi",
					},
				},
			}),
		},
		{
			// actions vs action
			expr: `event.data.name == "hi" && event.user.email == "what" && actions.first.data.ok == true && response.something == false`,
			in: NewData(map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"name":  "hi",
						"email": "some",
					},
					"user": map[string]interface{}{
						"external_id": 1,
						"email":       "test@egypt.lol",
					},
				},
				"actions": map[string]interface{}{
					"first": map[string]interface{}{
						"data": map[string]interface{}{
							"ok":      false,
							"another": 1274,
						},
					},
					"2": map[string]interface{}{
						"data": map[string]interface{}{
							"excluded": true,
						},
					},
				},
			}),
			expected: NewData(map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"name": "hi",
					},
					"user": map[string]interface{}{
						"email": "test@egypt.lol",
					},
				},
				"actions": map[string]interface{}{
					"first": map[string]interface{}{
						"data": map[string]interface{}{
							"ok": false,
						},
					},
				},
			}),
		},
		{
			// three present, different attrs
			expr: `event.data.name == "hi" && event.user.email == "what" && action.first.data.ok == true && response.something == false`,
			in: NewData(map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"name":  "hi",
						"email": "some",
					},
					"user": map[string]interface{}{
						"external_id": 1,
						"email":       "test@egypt.lol",
					},
				},
				"action": map[string]interface{}{
					"first": map[string]interface{}{
						"data": map[string]interface{}{
							"ok":      false,
							"another": 1274,
						},
					},
					"2": map[string]interface{}{
						"data": map[string]interface{}{
							"excluded": true,
						},
					},
				},
			}),
			expected: NewData(map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"name": "hi",
					},
					"user": map[string]interface{}{
						"email": "test@egypt.lol",
					},
				},
				"action": map[string]interface{}{
					"first": map[string]interface{}{
						"data": map[string]interface{}{
							"ok": false,
						},
					},
				},
			}),
		},
	}

	ctx := context.Background()

	for _, test := range tests {
		eval, err := NewExpressionEvaluator(ctx, test.expr)
		require.NoError(t, err)
		require.NotNil(t, test.in)
		actual := eval.FilteredAttributes(ctx, test.in)
		require.Equal(t, test.expected, actual)
	}
}

func BenchmarkEvaluate(b *testing.B) {
	expression := `["P0", "P1"].exists(p, p in event.data.tag) && lowercase(event.data.project) == "benchmark" && user.email.endsWith("example.net")`
	data := NewData(map[string]interface{}{
		"event": map[string]interface{}{
			"data": map[string]interface{}{
				"project": "Benchmark",
				"tag":     []string{"P0"},
			},
		},
		"user": map[string]interface{}{"email": "hi@example.net"},
	})
	ctx := context.Background()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		expr, err := NewBooleanEvaluator(ctx, expression)
		if err != nil {
			b.Fatalf("unknown error in benchmark: %s", err)
		}
		res, _, err := expr.Evaluate(ctx, data)
		if err != nil {
			b.Fatalf("unknown error in benchmark: %s", err)
		}
		if !res {
			b.Fatalf("benchmark expression failed")
		}
	}
}

func BenchmarkEvaluateParallel(b *testing.B) {
	expression := `["P0", "P1"].exists(p, p in event.data.tag) && lowercase(event.data.project) == "benchmark" && user.email.endsWith("example.net") && event.data.parallel != true`
	data := NewData(map[string]interface{}{
		"event": map[string]interface{}{
			"data": map[string]interface{}{
				"project": "Benchmark",
				"tag":     []string{"P0"},
			},
		},
		"user": map[string]interface{}{"email": "hi@example.net"},
	})
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			expr, _ := NewBooleanEvaluator(ctx, expression)
			res, _, err := expr.Evaluate(ctx, data)
			if err != nil {
				b.Fatalf("unknown error in benchmark: %s", err)
			}
			if !res {
				b.Fatalf("benchmark expression failed")
			}
		}
	})
}

func BenchmarkEvaluateRandomDataParallel(b *testing.B) {
	expression := `["P0", "P1"].exists(p, p in event.data.tag) && lowercase(event.data.project) == "benchmark" && user.email.endsWith("example.net") && event.data.random == true`
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			data := NewData(map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"project": "Benchmark",
						"tag":     []string{"P0"},
						"lol":     rand.Int(),
						"another": rand.Int(),
						"nested": map[string]interface{}{
							"yea": "plz",
						},
					},
				},
				"user": map[string]interface{}{"email": "hi@example.net"},
			})
			expr, err := NewExpressionEvaluator(ctx, expression)
			if err != nil {
				b.Fatalf("unknown error in benchmark: %s", err)
			}
			_, _, err = expr.Evaluate(ctx, data)
			if err != nil {
				b.Fatalf("unknown error in benchmark: %s", err)
			}
		}
	})
}

func BenchmarkEvaluateRandomExpressionParallel(b *testing.B) {
	data := NewData(map[string]interface{}{
		"event": map[string]interface{}{
			"data": map[string]interface{}{
				"project": "Benchmark",
				"tag":     []string{"P0"},
			},
		},
		"user": map[string]interface{}{"email": "hi@example.net"},
	})
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Create a new expression in parallel to ignore the cache.
			expression := fmt.Sprintf(`["P0", "P1"].exists(p, p in event.data.tag) && lowercase(event.data.project) == "benchmark" && user.email.endsWith("example.net") && 1 < %d`, rand.Int())
			expr, err := NewExpressionEvaluator(ctx, expression)
			if err != nil {
				b.Fatalf("unknown error in benchmark: %s", err)
			}
			_, _, err = expr.Evaluate(ctx, data)
			if err != nil {
				b.Fatalf("unknown error in benchmark: %s", err)
			}
		}
	})
}
