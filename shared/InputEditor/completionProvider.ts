import type monaco from "monaco-editor";
import { editor, languages } from "monaco-editor";
import { Data } from "./data";

export class CompletionProvider
  implements monaco.languages.CompletionItemProvider {
  // id stores the ID of the model that this completion data is for.  A model
  // represents the working data of a single monaco editor.
  //
  // This lets us show suggestions for the correct editor.
  _id: string = "";

  _list: Data;

  constructor(modelID: string, data: Data) {
    this._id = modelID;
    this._list = data;
  }

  provideCompletionItems = (
    model: editor.ITextModel,
    position: monaco.Position,
    _context: languages.CompletionContext,
    _token: monaco.CancellationToken
  ): languages.ProviderResult<languages.CompletionList> => {
    // There may be many instances of Monaco on the page.  We want to use
    // the correct data for the given editor model.
    if (model.id !== this._id) {
      return;
    }

    const pos = model.getValueInRange({
      startLineNumber: 1,
      startColumn: 1,
      endLineNumber: position.lineNumber,
      endColumn: position.column,
    });

    // Attempt to match template functions (`{{ event.data | `), prior
    // to matching templating data.
    let match = pos.match(/{{[^}}]*\|\s([^}}]*)$/);
    if (match) {
      return this.functionCompletionItems(model, position, match);
    }

    // Attempt to match template suggestions, matching any text after "{{",
    // with no closing token.
    match = pos.match(/{{[^}}]*?$/);
    if (match) {
      return this.templateCompletionItems(model, position, match);
    }

    return { suggestions: [] };
  };

  templateCompletionItems = (
    model: editor.ITextModel,
    position: monaco.Position,
    match: any
  ): languages.ProviderResult<languages.CompletionList> => {
    if (!match) {
      return { suggestions: [] };
    }

    // Find the very start word of the input.
    const word = model.getWordUntilPosition(position);

    // XXX: we can auto-show documentation:
    // https://stackoverflow.com/questions/54795603/always-show-the-show-more-section-in-monaco-editor

    // range represents where the search for the autocomplete starts from.  This is
    // the first templating character after "{{".
    const range = {
      startLineNumber: position.lineNumber,
      endLineNumber: position.lineNumber,
      startColumn: word.startColumn,
      endColumn: word.endColumn,
    };

    // match.index records where the first "{" character is, allowing us to properly format
    // templating with a single space ("{{ event.id }}") when inserting data.
    return this._list.suggestions(
      range,
      match.index ? match.index + 1 : undefined
    );
  };

  functionCompletionItems = (
    model: editor.ITextModel,
    position: monaco.Position,
    match: any
  ): languages.ProviderResult<languages.CompletionList> => {
    if (!match) {
      return { suggestions: [] };
    }

    // Find the very start word of the input.
    const word = model.getWordUntilPosition(position);
    const range = {
      startLineNumber: position.lineNumber,
      endLineNumber: position.lineNumber,
      startColumn: word.startColumn,
      endColumn: word.endColumn,
    };

    // match.index records where the first "{" character is, allowing us to properly format
    // templating with a single space ("{{ event.id }}") when inserting data.
    return {
      suggestions: [
        {
          label: "lower",
          detail: "Convert the data to lower case",
          insertText: "lower ",
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: "upper",
          detail: "Convert the data to upper case",
          insertText: "upper ",
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: "capitalize",
          detail: "Capitalize the first letter only",
          insertText: "capitalize ",
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: "title",
          detail: `Capitalize each word in a sentence`,
          insertText: `addslashes `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: `replace(from="", to="")`,
          detail: `Replace text.  Example: {{ name | replace(from="hello", to="hi")}}`,
          insertText: `replace(from="", to="") `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: "addslashes",
          detail: `Adds slashes before quotes`,
          insertText: `addslashes `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: "trim",
          detail: `Trims space around the string`,
          insertText: `trim `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: "trim_start",
          detail: `Trims space around the start of the string`,
          insertText: `trim_start `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: "trim_end",
          detail: `Trims space around the end of the string`,
          insertText: `trim_end `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: `truncate(length=10, end="")`,
          detail: `Truncate text`,
          insertText: `trim_end `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: `first`,
          detail: `Return the first element of an array`,
          insertText: `first `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: `last`,
          detail: `Return the last element of an array`,
          insertText: `last `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: `nth(n=2)`,
          detail: `Return the nth element of an array`,
          insertText: `nth(n=2) `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: `join(sep=", ")`,
          detail: `Joins an array with a string`,
          insertText: `join(sep=", ") `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: `length`,
          detail: `Returns the length of the array, string, or object`,
          insertText: `length `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: `reverse`,
          detail: `Reverses the string or array`,
          insertText: `reverse `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: `json_encode`,
          detail: `Transforms any value into a JSON representation`,
          insertText: `json_encode `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
        {
          label: `date(format="%Y-%m-%d %H:%M")`,
          detail: `Format a date`,
          insertText: `date(format="%Y-%m-%d %H:%M") `,
          kind: languages.CompletionItemKind.Function,
          range,
        },
      ],
    };
  };
}
