package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/cli/initialize"
	"github.com/inngest/inngest/pkg/function"
	"github.com/spf13/cobra"
)

func NewCmdTypes() *cobra.Command {
	root := &cobra.Command{
		Use:     "types",
		Short:   "Generate types from your Inngest Cloud account.",
		Example: "inngest types typescript",
	}

	typescript := &cobra.Command{
		Use:     "typescript",
		Aliases: []string{"ts"},
		Short:   "Generate TypeScript types in a .ts file",
		Run:     doTypescript,
	}

	root.AddCommand(typescript)

	root.PersistentFlags().StringP("output", "o", "./__generated__/inngest.ts", "Specify the location of the generated .ts file")

	return root
}

// Write the given `types` to the given `absPath`. Will try to create any
// required directories.
func writeTypes(types string, absPath string) error {
	// Try to create the folder and file with these TS types
	dirRequired := filepath.Dir(absPath)

	err := os.MkdirAll(dirRequired, 0755)
	if err != nil {
		return fmt.Errorf("couldn't create directory for output file; %w", err)
	}

	err = os.WriteFile(absPath, []byte(types), 0755)
	if err != nil {
		return fmt.Errorf("couldn't create output file; %w", err)
	}

	fmt.Println("Successfully created types at", absPath)

	return nil
}

// Checks that the given `relPath` is valid and contains the valid suffix
// depending on the types being generated. For example, it should be invalid to
// try to create TypeScript types as a `.go` file.
//
// `suffix` should be the file extension without the `.`, e.g. `ts` or `go`.
func checkOutputTarget(relPath string, suffix string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory; %w", err)
	}

	absPath := filepath.Join(cwd, relPath)
	if !strings.HasSuffix(absPath, suffix) {
		return "", fmt.Errorf("output file must be a .ts file")
	}

	return absPath, nil
}

func doTypescript(cmd *cobra.Command, args []string) {
	fmt.Println(cli.EnvString())

	absPath, err := checkOutputTarget(cmd.Flag("output").Value.String(), "ts")
	if err != nil {
		log.Fatalln("\n" + cli.RenderError(err.Error()) + "\n")
	}

	types, err := typescript(cmd, args)
	if err != nil {
		log.Fatalln("\n" + cli.RenderError(err.Error()) + "\n")
	}

	err = writeTypes(types, absPath)
	if err != nil {
		log.Fatalln("\n" + cli.RenderError(err.Error()) + "\n")
	}
}

// Fetch events and generate TypeScript types.
//
// TODO Extract fetching events here for easier reuse when adding other
// languages.
func typescript(cmd *cobra.Command, args []string) (string, error) {
	ctx := cmd.Context()

	// Try fetching events
	events, err := initialize.FetchEvents()
	if err != nil {
		if !strings.Contains(err.Error(), "not logged in") {
			return "", err
		}

		// If we got an error because we're not logged in, try logging in
		// and attempt once more
		DeviceAuth(ctx)
		events, err = initialize.FetchEvents()
		if err != nil {
			return "", err
		}
	}

	// Grab all events we've fetched and make sure they're ordered. We want to
	// preserve this order so we can perform CI checks in the future, comparing
	// Inngest Cloud types with our local types.
	unorderedEvents := events.All()
	eventKeys := make([]string, 0, len(unorderedEvents))
	for k := range unorderedEvents {
		eventKeys = append(eventKeys, k)
	}
	sort.Strings(eventKeys)

	// Create a map of event IDs to the name of the type in the TS file. We'll use
	// this to create a larger catch-all type later for `new Inngest()`.
	eventNames := make(map[string]string)
	b := &strings.Builder{}
	b.WriteString(`// Generated via inngest types` + "\n\n")

	for _, eventId := range eventKeys {
		event := unorderedEvents[eventId]

		if event.Event.Versions[0].CueType == "" {
			continue
		}

		et := function.EventTrigger{
			Event: eventId,
			Definition: &function.EventDefinition{
				Format: function.FormatCue,
				Synced: true,
				Def:    event.Event.Versions[0].CueType,
			},
		}

		eventName := et.TitleName()
		eventNames[eventId] = eventName

		ts, err := et.Definition.Typescript(ctx)
		if err != nil {
			continue
		}

		// Replace "interface Event" and instead name the event type explicitly
		ts = strings.Replace(ts, "interface Event", fmt.Sprintf("type %s =", eventName), 1)

		// Replace any instance of `name: string;` with the actual const name of the
		// event
		prefix := "\n  name: "
		suffix := ";"
		ts = strings.Replace(ts, fmt.Sprintf("%sstring%s", prefix, suffix), fmt.Sprintf("%s\"%s\"%s", prefix, eventId, suffix), 1)

		// Replace any instance of `{}` with `Record<string, never>`. TS types
		// declare `{}` as "any object", which means it could be pretty much
		// anything in JS.
		ts = strings.Replace(ts, "{}", "Record<string, never>", -1)

		b.WriteString(ts + "\n")
	}

	// Add a type to allow users to define custom types
	b.WriteString("type CustomEvent = {\n")
	b.WriteString("  data: Record<string, any>;\n")
	b.WriteString("  user?: Record<string, any>;\n")
	b.WriteString("};\n\n")

	// Create a catch-all type that we'll use when creating clients
	b.WriteString("type GeneratedEvents = Readonly<{\n")
	for eventId, tsEventName := range eventNames {
		b.WriteString(fmt.Sprintf("  \"%s\": Readonly<%s>;\n", eventId, tsEventName))
	}
	b.WriteString("}>;\n\n")

	// Create the exported `Events` type that can take optional custom events
	b.WriteString("/**\n")
	b.WriteString(" * Events generated from real data in your Inngest Cloud account. Can be passed\n")
	b.WriteString(" * an object containing custom events if you wisht to send events not yet in\n")
	b.WriteString(" * your ecosystem.\n")
	b.WriteString(" *\n")
	b.WriteString(" * ```ts\n")
	b.WriteString(" * const inngest = new Inngest<\n")
	b.WriteString(" *   Events<{\n")
	b.WriteString(" *     \"app/user.created\": {\n")
	b.WriteString(" *       data: { id: string; email: string };\n")
	b.WriteString(" *     };\n")
	b.WriteString(" *   }>\n")
	b.WriteString(" * >({ name: \"My App\" });\n")
	b.WriteString(" * ```\n")
	b.WriteString(" */\n")
	b.WriteString("// eslint-disable-next-line @typescript-eslint/ban-types\n")
	b.WriteString("export type Events<CustomEvents extends Record<string, CustomEvent> = {}> =\n")
	b.WriteString("  Readonly<Omit<CustomEvents, keyof GeneratedEvents> & GeneratedEvents>;")

	return b.String(), nil
}
