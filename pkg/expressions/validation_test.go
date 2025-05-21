package expressions

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func Test_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expr        string
		policy      *ValidationPolicy
		expectValid bool
	}{
		{
			expr:        "event.data.foo == 1",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event.data.foo == 1",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        "event.name.endsWith(\"foo\")",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "event.name.endsWith(\"foo\")",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "has(event.data.isPreview)",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "has(event.data.isPreview)",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        `"lol" == event.data || event.name == "foo" || (async.source == 'test' || async.source == 'google') && date(async.next_visit) > now_plus("24h") && async.email == 'test@example.com'`,
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "event.data.action == 'opened'",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event.data.action == 'opened'",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        "event.data.name",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event.data.name",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        "5 + 4",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "5 + 4",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "event.data.name + '-' + event.data.number + '-' + event.data.bool + '-' + event.data.list + '-' + event.data.obj + '-' + event.data.missing",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event.data.name + '-' + event.data.number + '-' + event.data.bool + '-' + event.data.list + '-' + event.data.obj + '-' + event.data.missing",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "uppercase(event.data.name) + ' is my name'",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "uppercase(event.data.name) + ' is my name'",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        `b64decode("aGkgdGhlcmUh") + " - b64 data"`,
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        `b64decode("aGkgdGhlcmUh") + " - b64 data"`,
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        `json_parse(b64decode(event.data.encoded)).nested.data`,
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        `json_parse(b64decode(event.data.encoded)).nested.data`,
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "null > 0",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "null > 0",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        "size(event.data.nope) > 0",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "size(event.data.nope) > 0",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "event.data.float <= null && event.data.int < null",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event.data.float <= null && event.data.int < null",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        "event.data.int > 3.141 && event.data.float <= 2",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event.data.int > 3.141 && event.data.float <= 2",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        `event.first.foo == 'test' && event.second.bar != true && event.second.baz != true`,
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        `event.first.foo == 'test' && event.second.bar != true && event.second.baz != true`,
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        `async.first.foo != 'test' && (async.first.bar == true || async.second.baz == true)`,
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        `async.first.foo != 'test' && (async.first.bar == true || async.second.baz == true)`,
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        "event.data.foo != '' && 'item' in event.data.issue.fields.tags",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event.data.foo != '' && 'item' in event.data.issue.fields.tags",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        "event['data'].foo != '' && event['data'].bar != true",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event['data'].foo != '' && event['data'].bar != true",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        "event.data.test == true && event.data.issue.field.another != 'another'",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event.data.test == true && event.data.issue.field.another != 'another'",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        `event.nonexistent.exists(x, x == 'a')`,
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        `event.nonexistent.exists(x, x == 'a')`,
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        `event.data.title.matches('^[a-z\\d]{0,10}$')`,
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        `event.data.title.matches('^[a-z\\d]{0,10}$')`,
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "event.data.some.unknown.contains(event.data.title)",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event.data.some.unknown.contains(event.data.title)",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "size(event.data.action) > 0",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "size(event.data.action) > 0",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "int(async.status) == 200",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "int(async.status) == 200",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        "event.data.tags.exists(x, x == 'd') == false",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event.data.tags.exists(x, x == 'd') == false",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "event.data.issue in ['Bug', 'Issue', 'Epic'] == true",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "event.data.issue in ['Bug', 'Issue', 'Epic'] == true",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: true,
		},
		{
			expr:        "[1, 2, 3].all(x, x > 0) && event.status == '200'",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "[1, 2, 3].all(x, x > 0) && event.status == '200'",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
		{
			expr:        "date(event.from) + duration('6h35m10s') == date('2020-01-01T18:35:10')",
			policy:      nil,
			expectValid: true,
		},
		{
			expr:        "date(event.from) + duration('6h35m10s') == date('2020-01-01T18:35:10')",
			policy:      DefaultRestrictiveValidationPolicy(),
			expectValid: false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d (expectValid: %v)", i, tt.expectValid), func(t *testing.T) {
			t.Parallel()

			err := Validate(context.Background(), tt.policy, tt.expr)
			if err != nil && tt.expectValid {
				t.Errorf("expected to be valid; got: %v\nexpr: %s", err, tt.expr)
			} else if err == nil && !tt.expectValid {
				t.Errorf("expected to be invalid; got: validation passed\nexpr: %s", tt.expr)
			} else if err != nil && !errors.Is(err, ErrValidationFailed) {
				t.Errorf("expected validation error; got: %v\nexpr: %s", err, tt.expr)
			} else if err != nil {
				t.Logf("[info] validation error was: %v", err)
			}
		})
	}
}
