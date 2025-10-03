'use client';

import type { SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

type Editor = Parameters<SQLEditorMountCallback>[0];
type Monaco = Parameters<SQLEditorMountCallback>[1];
type Model = NonNullable<ReturnType<Editor['getModel']>>;
type Marker = Parameters<Monaco['editor']['setModelMarkers']>[2][number];

const OWNER = 'template-vars' as const;

// Matches "{{ <content> }}" where:
// - at least one space follows the opening braces
// - <content> contains at least one letter (A–Z or a–z)
// - at least one space precedes the closing braces
// This keeps markers limited to meaningful template variables and avoids noisy matches.
const PATTERN = /\{\{\s+(?=[^}]*[A-Za-z])[^}]*\s+\}\}/g;

const NOOP = () => {};

export function markTemplateVars(editor: Editor, monaco: Monaco) {
  const model = editor.getModel();
  if (model === null) return { dispose: NOOP };

  updateTemplateVarMarkers(model, monaco);

  const disposable = model.onDidChangeContent(() => {
    updateTemplateVarMarkers(model, monaco);
  });

  return {
    dispose: () => {
      clearTemplateVarMarkers(model, monaco);
      disposable.dispose();
    },
  };
}

function updateTemplateVarMarkers(model: Model, monaco: Monaco) {
  const matches = identifyTemplateVars(model.getValue());
  const ranges = getTemplateVarsRanges(model, matches);
  assignTemplateVarsMarkers(monaco, model, ranges);
}

function clearTemplateVarMarkers(model: Model, monaco: Monaco) {
  monaco.editor.setModelMarkers(model, OWNER, []);
}

type TemplateVarMatch = Readonly<{ startIndex: number; endIndex: number }>;

function identifyTemplateVars(text: string) {
  const results: TemplateVarMatch[] = [];

  for (const match of text.matchAll(PATTERN)) {
    const idx = match.index;
    if (typeof idx !== 'number') continue;

    results.push({
      startIndex: idx,
      endIndex: idx + match[0].length,
    });
  }

  return results;
}

type MarkerRange = Pick<Marker, 'startLineNumber' | 'startColumn' | 'endLineNumber' | 'endColumn'>;

function getTemplateVarsRanges(model: Model, matches: TemplateVarMatch[]) {
  return matches.map((m): MarkerRange => {
    const start = model.getPositionAt(m.startIndex);
    const end = model.getPositionAt(m.endIndex);

    return {
      startLineNumber: start.lineNumber,
      startColumn: start.column,
      endLineNumber: end.lineNumber,
      endColumn: end.column,
    };
  });
}

function assignTemplateVarsMarkers(monaco: Monaco, model: Model, ranges: MarkerRange[]) {
  const markers: Marker[] = ranges.map((range) => ({
    ...range,
    message: 'Provide a value for this template variable.',
    severity: monaco.MarkerSeverity.Info,
  }));

  monaco.editor.setModelMarkers(model, OWNER, markers);
}
