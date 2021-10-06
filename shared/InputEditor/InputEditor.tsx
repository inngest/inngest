import React, { useState, useEffect, useRef } from "react";
import styled from "@emotion/styled";
import { ErrorBoundary } from "react-error-boundary";
import Editor, { Monaco } from "@monaco-editor/react";
import type monaco from "monaco-editor";
import { CompletionProvider } from "./completionProvider";
import { options } from "./options";
import { CompletionItemSubset, Data } from "./data";

// monaco uses process.env to figure out whether to load, and env must be a map.
// One of our other dependencies adds the `process` global without env.
// @ts-ignore
window.process.env = {};

export type Props = {
  kind?: KindStrings;
  language?: string;
  placeholder?: string;
  disabled?: boolean;

  onChange?: (value: string) => void;
  value?: string;

  recommendations?: Array<string | CompletionItemSubset>;
};

export enum Kinds {
  input,
  code,
  textarea,
}

export type KindStrings = keyof typeof Kinds;

// InputEditor creates an input editor for use within workflow configuration.  The editor's
// primary aim is to **make templatig easy**.
//
// We do this by:
//
// 1. Using Monaco under the hood.
// 2. Creating a "completion item provider" for intellisense that matches on templating (`{{`)
// 3. Automatically inferring the properties available for the current editor/action
// 4. Showing which items are available, and what they mean.
export const InputEditor: React.FC<Props> = (props) => {
  const ref = useRef<monaco.editor.IStandaloneCodeEditor | null>(null);
  const monacoref = useRef<Monaco | null>(null);
  const completion = useRef<monaco.IDisposable | null>(null);
  const highlights = useRef<string[]>([]);
  const [placeholder, setPlaceholder] = useState(!props.value);

  const language = props.language || "inngest";

  // When recommendations change, update the autocomplete suggestions.  This may
  // not be present when the component has mounted, or the example event may change.
  const setAutocompletes = () => {
    const model = ref?.current?.getModel();

    // There's a race condition where ref.current isn't set, as that happens on
    // mount which may take > 1 second.
    if (
      !model &&
      Array.isArray(props.recommendations) &&
      props.recommendations.length > 0
    ) {
      // Retry.
      console.warn("retrying autocomplete");
      window.setTimeout(setAutocompletes, 100);
    }

    if (!model || !monacoref.current) return;

    // Ensure we have a language registered for our autocomplete
    !props.language && monacoref.current.languages.register({ id: "inngest" });

    // When recommendations change, update the autocomplete suggestions.  This may
    // not be present when the component has mounted, or the example event may change.
    // completion.current && completion.current.dispose();
    completion?.current?.dispose();
    completion.current = monacoref.current.languages.registerCompletionItemProvider(
      language,
      new CompletionProvider(
        model.id,
        new Data(
          props.recommendations || ([] as Array<string | CompletionItemSubset>)
        )
      )
    );
  };

  const didMount = (
    editor: monaco.editor.IStandaloneCodeEditor,
    monaco: Monaco
  ) => {
    ref.current = editor;
    monacoref.current = monaco;

    // language: https://microsoft.github.io/monaco-editor/monarch.html
    const model = editor.getModel();
    if (!model) return;

    !props.language && monaco.languages.register({ id: "inngest" });

    editor.onMouseMove((_e) => {
      // TODO: Detect target and see if we're hovering on a template decoration
    });

    // Once mounted, calculate where all the template boxes are on the first render.
    calculateTemplateBoxes();
  };

  useEffect(() => {
    setAutocompletes();
  }, [props.recommendations]);

  useEffect(() => {
    // When this component unmounts, dispose of the completion data for the text editor.
    return () => {
      completion.current && completion.current.dispose();
    };
  }, []);

  useEffect(() => {
    if (!ref.current) return;
    ref.current.updateOptions({ readOnly: !!props.disabled });
  }, [props.disabled]);

  const calculateTemplateBoxes = () => {
    if (!ref.current) return;
    const m = ref.current.getModel();
    if (!m) return;

    // Find all matches for {{ templating }} tokens, then highlight them.
    const templates = m.findMatches(
      "\\{\\{[^\\}\\}]*\\}\\}",
      true,
      true,
      false,
      null,
      true
    );

    if (!templates) {
      highlights.current = ref.current.deltaDecorations(highlights.current, []);
      return;
    }

    // For each matching bracket set, add decorations which highlight the current word.
    const decs = templates
      .map((t) => {
        return [
          // The first decoration is the clickable outline box.  The second is the text background color.
          // When clicking this, we want to set the cursor position to the beginning of the range,
          // then set the selection to the template content.
          {
            range: t.range,
            options: {
              before: { content: "" },
              after: { content: "" },
              className: "template",
              hoverMessage: {
                value: "This will be replaced with data when the action runs",
              },
              afterContentClassName: "template-post",
              beforeContentClassName:
                t.range.startColumn === 1
                  ? "template-pre-none"
                  : "template-pre",
            },
          },
          {
            range: t.range,
            options: { className: "template-color" },
          },
        ];
      })
      .flat();
    highlights.current = ref.current.deltaDecorations(highlights.current, decs);
  };

  return (
    <ErrorBoundary FallbackComponent={() => <Input {...props} />}>
      <Wrapper
        className={`editor ${props.kind}`}
        key={props.disabled ? "read-only" : "write"}
      >
        <Placeholder className={placeholder ? `show ${props.kind}` : "hide"}>
          {props.placeholder}
        </Placeholder>
        <Editor
          defaultValue={props.value}
          onMount={didMount}
          language={props.language || "inngest"}
          loading={"Loading..."}
          options={{
            ...options[props.kind || "input"],
            readOnly: !!props.disabled,
          }}
          onChange={() => {
            if (!!props.disabled) return;
            if (!ref.current) return;

            const m = ref.current.getModel();
            if (!m) return;

            setPlaceholder(!m.getValue());

            props.onChange && props.onChange(m.getValue());

            calculateTemplateBoxes();
          }}
        />
      </Wrapper>
    </ErrorBoundary>
  );
};

const Input: React.FC<Props> = (props) => {
  console.warn("rendering fallback");
  return (
    <input
      placeholder={props.placeholder}
      onChange={(e) => props.onChange && props.onChange(e.target.value)}
      disabled={props.disabled}
      value={props.value}
    />
  );
};

InputEditor.defaultProps = {
  kind: "input",
};

export default InputEditor;

const classes: { [key in keyof typeof Kinds]: any } = {
  input: ".input",
  code: ".code",
  textarea: ".textarea",
};

const Wrapper = styled.div`
  background: #fff;
  border-radius: 2px;
  border: 1px solid #eee;
  transition: all 0.3s;
  box-shadow: 0 1px 0px rgba(0, 0, 0, 0.05), 0 1px 3px rgba(0, 0, 0, 0.03);
  width: 100%;
  position: relative;

  &:focus {
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
    border: 1px solid #c5c5c5;
  }

  /* Renders a box around templates in our editor */
  .template,
  .template-color {
    border-radius: 2px;
    padding: 4px 9px 4px 10px;
    margin: -4px 0 0 -10px;
    box-sizing: content-box;
  }

  .template {
    border: 1px solid transparent;
    box-shadow: 0 0 12px rgba(0, 0, 0, 0.08);
    cursor: pointer;
  }

  .template-color {
    border: 1px solid #e8e8e6;
    background: rgb(249, 243, 230);
    /* this ensures we can see text selections */
    z-index: -1;
  }

  .template-pre {
    display: inline-block;
    width: 18px;
  }
  .template-pre-none {
    display: inline-block;
    width: 10px;
  }
  .template-post {
    display: inline-block;
    width: 17px;
  }

  &${classes.input} {
    box-sizing: border-box;
    height: 2.8rem;
    line-height: 1rem;
  }

  &${classes.textarea}, &${classes.code} {
    box-sizing: border-box;
    height: 10rem;
    line-height: 1rem;
  }

  &${classes.code} {
    padding-left: 10px;
  }
`;

const Placeholder = styled.span`
  font-size: 14px;
  position: absolute;
  opacity: 0.5;
  z-index: 10;
  top: 11px;
  left: 12px;
  &.hide {
    display: none;
  }

  &${classes.textarea}, &${classes.code} {
    font-family: monospace;
  }
`;
