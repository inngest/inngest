package scaffold

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gosimple/slug"
	"github.com/inngest/cuetypescript"
	"github.com/inngest/inngest/pkg/function"
	"github.com/karrick/godirwalk"
)

type Template struct {
	Name        string
	Description string
	Targets     []string
	PostSetup   string
	root        string
}

type tplData struct {
	ID            string
	Name          string
	QuotedName    string
	SlugName      string
	EventTriggers []*function.EventTrigger
}

func (t Template) TemplatedPostSetup(f function.Function) string {
	tpl, err := template.New("postsetup").Parse(string(t.PostSetup))
	if err != nil {
		return t.PostSetup
	}

	data := map[string]string{
		"id":   f.ID,
		"name": f.Name,
		"slug": f.Slug(),
		"dir":  f.Slug(),
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, data); err != nil {
		return t.PostSetup
	}

	return buf.String()
}

func (t Template) Root(f function.Function) string {
	if f.Dir() != "" {
		// This already exists;  return the function dir.
		return f.Dir()
	}

	// We're making a new dir.
	dirname := f.Slug()
	relative := "./" + dirname
	root, _ := filepath.Abs(relative)
	return root
}

// RenderNew creates a new function when no files exist.
func (t Template) RenderNew(ctx context.Context, f function.Function) error {
	dirname := f.Slug()
	root := t.Root(f)

	if _, err := os.Stat(root); err == nil {
		return fmt.Errorf("%s already exists", dirname)
	}

	if err := os.Mkdir(root, 0755); err != nil {
		return fmt.Errorf("error creating function directory: %w", err)
	}

	for _, s := range f.Steps {
		if err := t.RenderStep(ctx, f, s); err != nil {
			return err
		}
	}

	return f.WriteToDisk(ctx)
}

// Render renders the template and all files into the folder specified by function.
func (t Template) RenderStep(ctx context.Context, f function.Function, step function.Step) error {
	root := t.Root(f)

	stepDir, err := function.PathName(ctx, step.Path)
	if err != nil {
		return err
	}

	data := tplData{
		ID:         f.ID,
		Name:       f.Name,
		QuotedName: strings.ReplaceAll(f.Name, `"`, `\"`),
		SlugName:   slug.Make(f.Name),
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
					ts, err := t.Definition.Typescript(ctx)
					if err != nil {
						continue
					}

					// Write the type, replacing "Event" with the event name.
					ts = strings.Replace(ts, "interface Event", fmt.Sprintf("interface %s", t.TitleName()), 1)
					names = append(names, t.TitleName())

					b.WriteString(ts + "\n")
				}

				if len(names) == 0 {
					return "export type EventTriggers = { [key: string]: any };"
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

	if err := upsertDir(filepath.Join(root, "steps")); err != nil {
		return fmt.Errorf("error making steps directory: %w", err)
	}

	stepRoot := filepath.Join(root, stepDir)
	if err := os.MkdirAll(stepRoot, 0755); err != nil {
		return fmt.Errorf("error making step directory: %w", err)
	}

	// Clone the template dir and run templating on every file.
	if t.root != "" {
		err := godirwalk.Walk(t.root, &godirwalk.Options{
			Callback: func(absPath string, d *godirwalk.Dirent) error {
				path, err := filepath.Rel(t.root, absPath)
				if err != nil {
					return err
				}

				if path == "." {
					return nil
				}

				if d.IsDir() {
					if err := os.Mkdir(filepath.Join(stepRoot, path), 0755); err != nil {
						return err
					}
					return nil
				}

				file, err := os.Open(absPath)
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

				return os.WriteFile(filepath.Join(stepRoot, path), buf.Bytes(), 0644)
			},
			Unsorted: false,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func upsertDir(path string) error {
	if exists(path) {
		return nil
	}
	return os.MkdirAll(path, 0755)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
