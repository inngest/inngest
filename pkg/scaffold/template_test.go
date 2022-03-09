package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/inngest/inngestctl/pkg/function"
	"github.com/stretchr/testify/require"
)

func TestTemplateRenderTypescript(t *testing.T) {
	mapping, err := parse(context.Background(), fixtures)
	require.NoError(t, err)

	tpl := mapping.Languages["typescript"][0]

	f, err := function.New()
	require.NoError(t, err)
	f.Name = "template render"
	f.Triggers = append(
		f.Triggers,
		function.Trigger{
			EventTrigger: &function.EventTrigger{
				Event: "first",
				Definition: &function.EventDefinition{
					Format: function.FormatCue,
					Def: `{
  name: string
  data: {
    id:   int
    name: string
    by:   string
    at:   string
  }
  user: {
    email: string
  }
  ts: int
}`,
				},
			},
		},
		function.Trigger{
			EventTrigger: &function.EventTrigger{
				Event: "second.event",
				Definition: &function.EventDefinition{
					Format: function.FormatCue,
					Def: `{
  name: string
  data: {
    account: string
  }
  user: {
    email: string
  }
  ts: int
}`,
				},
			},
		},
	)

	root, _ := filepath.Abs("./" + f.Slug())
	os.RemoveAll(root)

	err = tpl.Render(*f)
	require.NoError(t, err)

	// Expect "types.ts" to contain genned types.
	byt, err := os.ReadFile(filepath.Join(root, "types.ts"))
	require.NoError(t, err)
	require.EqualValues(t, expectedTypes, string(byt))
}

var expectedTypes = `// Generated via inngest init

export interface First {
  name: string;
  data: {
    id: number;
    name: string;
    by: string;
    at: string;
  };
  user: {
    email: string;
  };
  ts: number;
};

export interface SecondEvent {
  name: string;
  data: {
    account: string;
  };
  user: {
    email: string;
  };
  ts: number;
};

export type EventTriggers = First | SecondEvent;

export type Args = {
  event: EventTriggers;
  actions: {
    [clientID: string]: any
  };
};

`
