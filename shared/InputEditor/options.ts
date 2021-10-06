import { editor } from "monaco-editor";
import { Kinds } from "./InputEditor";

// inputOptions define options which make the editor look like a standard
// input box.
const inputOptions: editor.IStandaloneEditorConstructionOptions = {
  lineNumbers: "off",
  minimap: { enabled: false },
  scrollbar: { vertical: "hidden" },

  fontFamily: "inherit",
  fontSize: 14,

  // remove the right hand border visuals
  hideCursorInOverviewRuler: true,
  overviewRulerBorder: false,
  overviewRulerLanes: 0,
  renderLineHighlight: "none",

  // scrollBeyondLastLine prevents the editor from rendering a new blank line
  // on single input texts, which would shift the input content on click.
  scrollBeyondLastLine: false,

  codeLens: false,

  // remove the left hand border visuals
  glyphMargin: false,
  folding: false,
  lineNumbersMinChars: 0,

  // Padding:
  lineDecorationsWidth: 14, // left/right
  padding: { bottom: 11, top: 11 },
};

const monospace = {
  fontFamily: "monospace",
  fontSize: 13,
};

const textareaOptions: editor.IStandaloneEditorConstructionOptions = {
  ...inputOptions,
  ...monospace,
  wordWrap: "on",
};

// codeOptions define options for code editors.
const codeOptions: editor.IStandaloneEditorConstructionOptions = {
  ...inputOptions,
  ...monospace,

  lineNumbers: "on",
  wordWrap: "on",

  glyphMargin: false,
  folding: true,
  lineNumbersMinChars: 0,

  // Padding:
  lineDecorationsWidth: 8, // left/right
  padding: { bottom: 11, top: 11 },
};

export const options: {
  [key in keyof typeof Kinds]: editor.IStandaloneEditorConstructionOptions;
} = {
  input: inputOptions,
  textarea: textareaOptions,
  code: codeOptions,
};
