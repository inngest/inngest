package scaffold

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/inngest/cuetypescript"
	"github.com/inngest/inngestctl/pkg/function"
)

type Template struct {
	Name        string
	Description string
	Targets     []string
	PostSetup   string

	FS fs.FS
}

type tplData struct {
	ID            string
	Name          string
	QuotedName    string
	EventTriggers []*function.EventTrigger
}

// Render renders the template and all files into the folder specified by function.
func (t Template) Render(f function.Function) error {
	dirname := f.Slug()
	root, _ := filepath.Abs("./" + dirname)

	if _, err := os.Stat(root); err == nil {
		return fmt.Errorf("%s already exists", dirname)
	}

	if err := os.Mkdir(root, 0755); err != nil {
		return fmt.Errorf("error creating function directory: %w", err)
	}

	data := tplData{
		ID:         f.ID,
		Name:       f.Name,
		QuotedName: strings.ReplaceAll(f.Name, `"`, `\"`),
	}

	for _, t := range f.Triggers {
		if t.EventTrigger == nil {
			continue
		}
		data.EventTriggers = append(data.EventTriggers, t.EventTrigger)
	}

	funcMap := template.FuncMap{
		// The name "title" is what the function will be called in the template text.
		"EventTypes": func(language string) string {
			switch language {
			case "typescript":
				// Store all event names.
				names := []string{}

				b := &strings.Builder{}
				b.WriteString(`// Generated via inngest init` + "\n\n")

				for _, t := range data.EventTriggers {
					if t.Definition == nil {
						continue
					}
					ts, err := t.Definition.Typescript()
					if err != nil {
						continue
					}

					// Write the type, replacing "Event" with the event name.
					ts = strings.Replace(ts, "interface Event", fmt.Sprintf("interface %s", t.TitleName()), 1)
					names = append(names, t.TitleName())

					b.WriteString(ts + "\n")
				}

				if len(names) == 0 {
					return ""
				}

				// Write an enum which joins all event triggers.
				b.WriteString(fmt.Sprintf("export type EventTriggers = %s;", strings.Join(names, " | ")))
				return b.String()
			default:
				return fmt.Sprintf("unsupported language %s", language)
			}
		},
		"typescript": func(e *function.EventDefinition) string {
			if e == nil {
				return "any;"
			}
			str, _ := cuetypescript.MarshalString(e.Def)
			return str
		},
	}

	// Clone the template dir and run templating on every file.
	if t.FS != nil {
		err := fs.WalkDir(t.FS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if path == "." {
				return nil
			}

			if d.IsDir() {
				if err := os.Mkdir(filepath.Join(root, path), 0755); err != nil {
					return err
				}
				return nil
			}

			file, err := t.FS.Open(path)
			if err != nil {
				return err
			}
			contents, err := io.ReadAll(file)
			if err != nil {
				return err
			}

			tpl, err := template.New(path).Funcs(funcMap).Parse(string(contents))
			if err != nil {
				return err
			}

			buf := &bytes.Buffer{}
			if err := tpl.Execute(buf, data); err != nil {
				return err
			}

			// TODO: Template contents.
			return os.WriteFile(filepath.Join(root, path), buf.Bytes(), 0644)
		})
		if err != nil {
			return err
		}
	}

	// Once complete, state should contain everything we need to create our
	// function file.
	byt, err := function.MarshalJSON(f)
	if err != nil {
		return fmt.Errorf("error creating JSON: %s", err)
	}

	if err := os.WriteFile(filepath.Join(root, "inngest.json"), byt, 0644); err != nil {
		return fmt.Errorf("Error writing inngest.json: %s", err)
	}

	return nil
}
