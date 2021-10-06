import type monaco from "monaco-editor";
import { languages } from "monaco-editor";

interface CompletionList {
  suggestions(
    r: monaco.Range,
    bracketPos: number
  ): monaco.languages.CompletionList;
}

// Data represents all completion data for a given monaco editor.
// It is generated
export class Data implements CompletionList {
  _list: CompletionItemSubset[];

  constructor(list: Array<string | CompletionItemSubset>) {
    this._list = list.map((l) => {
      if (typeof l === "string") {
        return {
          label: l,
          insertText: `${l}`,
          kind: languages.CompletionItemKind.Field,
        };
      }
      return l;
    });
  }

  suggestions = (
    r: monaco.IRange,
    bracketPos?: number
  ): monaco.languages.CompletionList => {
    return {
      suggestions: this._list.map((i) => ({
        ...i,
        range: r,
        // Replace the original starting brackets.
        // TODO: Add additional ending brackets if they don't exist immediately
        // after the current cursor (not including spaces).
        additionalTextEdits:
          bracketPos === undefined
            ? []
            : [
                {
                  range: {
                    ...r,
                    startColumn: bracketPos,
                  },
                  text: null,
                },
              ],
      })),
    };
  };
}

export interface CompletionItemSubset {
  label: string | monaco.languages.CompletionItemLabel;
  kind: monaco.languages.CompletionItemKind;
  insertText: string;
  tags?: ReadonlyArray<monaco.languages.CompletionItemTag>;
  detail?: string;
  documentation?: string | monaco.IMarkdownString;
  sortText?: string;
  filterText?: string;
  preselect?: boolean;
}
