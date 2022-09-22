package commands

import (
	"bytes"
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
	root.PersistentFlags().BoolP("check", "c", false, "Compare found types with an existing generated file defined by --output, failing if they're different. Useful for CI to check you are up to date")

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

	shouldCheck := cmd.Flag("check").Value.String() == "true"
	if shouldCheck {
		// load file, check if the same
		currFile, err := os.ReadFile(absPath)
		if err != nil {
			log.Fatalln("\n" + cli.RenderError("Failed to read existing file; make sure to specify your types file using --output") + "\n")
		}

		diff := bytes.Compare(currFile, []byte(types))
		if diff > 0 {
			log.Fatalln("\n" + cli.RenderError("Local types are different to the latest types from Inngest Cloud. Consider regenerating your local types."))
		}

		fmt.Println("Local types are in sync with the latest types from Inngest Cloud.")
		return
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

		b.WriteString(ts + "\n")
	}

	// Create a catch-all type that we'll use when creating clients
	b.WriteString("export type Events = {\n")
	for eventId, tsEventName := range eventNames {
		b.WriteString(fmt.Sprintf("  \"%s\": %s;\n", eventId, tsEventName))
	}
	b.WriteString("};\n")

	return b.String(), nil
}
