import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Button } from '@inngest/components/Button';
import { CopyButton } from '@inngest/components/CopyButton';
import { maxRenderedOutputSizeBytes } from '@inngest/components/constants';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { IconOverflowText } from '@inngest/components/icons/OverflowText';
import { IconWrapText } from '@inngest/components/icons/WrapText';
import { cn } from '@inngest/components/utils/classNames';
import { FONT, LINE_HEIGHT, createColors, createRules } from '@inngest/components/utils/monaco';
import Editor, { useMonaco, type Monaco } from '@monaco-editor/react';
import {
  RiCollapseDiagonalLine,
  RiDownload2Line,
  RiEdit2Line,
  RiExpandDiagonalLine,
  RiInformationLine,
} from '@remixicon/react';
import type { editor } from 'monaco-editor';
import { JSONTree } from 'react-json-tree';
import useLocalStorage from 'react-use/esm/useLocalStorage';

import { Alert } from '../Alert';
import { Fullscreen } from '../Fullscreen/Fullscreen';
import { Pill } from '../Pill';
import SegmentedControl from '../SegmentedControl/SegmentedControl';
import { Skeleton } from '../Skeleton';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';
import { jsonTreeTheme } from '../utils/jsonTree';
import { isDark } from '../utils/theme';

// JSON field path display: shows the full path (e.g. "data.users[0].name") at the bottom
// of the editor when the cursor is on a line, similar to the fx terminal tool.
const EMPTY_INPUT = JSON.stringify({ data: {} }, null, 2);

export type CodeBlockAction = {
  label: string;
  title?: string;
  icon?: ReactNode;
  onClick: () => void;
  disabled?: boolean;
};

interface CodeBlockProps {
  className?: string;
  header?: {
    title?: string;
    status?: 'success' | 'error';
  };
  tab: {
    content: string;
    readOnly?: boolean;
    language?: string;
    handleChange?: (value: string) => void;
  };
  actions?: CodeBlockAction[];
  allowFullScreen?: boolean;
  enableTreeView?: boolean;
  loading?: boolean;
  scrollbarOptions?: editor.IEditorScrollbarOptions;
  // Plain-click links overlaid on the raw Monaco text view (opened in a
  // new tab) -- e.g. turning known ID strings into links to their run/
  // event/etc. page -- while the text itself still renders exactly as it
  // does today (no separate tree/list view). Deliberately not Monaco's
  // built-in registerLinkProvider: that only activates on Cmd/Ctrl+click
  // (hardcoded into Monaco's own LinkDetector, not configurable), so this
  // instead applies its own underline decorations and a plain mousedown
  // handler that navigates directly -- see applyMonacoLinkDecorations.
  // Each entry's `text` is located as a literal substring of the editor's
  // own live text -- the caller is responsible for matching whatever
  // quoting/formatting `tab.content` actually renders (e.g.
  // JSON.stringify(value) for one element of a JSON array, or the bare
  // value itself for a plaintext scalar cell). Entries may be given in
  // any order (matching tolerates e.g. an object field appearing before
  // or after a sibling field, whichever order the driver/JSON serializer
  // actually produced).
  monacoLinks?: { text: string; href: string }[];
}

function buildJsonPath(keyPath: readonly (string | number)[]): string {
  // keyPath is ordered from leaf to root, so reverse it and skip "root"
  const parts = [...keyPath].reverse();
  let result = '';
  for (const part of parts) {
    if (typeof part === 'number') {
      result += `[${part}]`;
    } else {
      result += result ? `.${part}` : part;
    }
  }
  return result;
}

type ActiveMonacoLink = { range: InstanceType<Monaco['Range']>; url: string };

// Only http(s) navigation is ever legitimate for a monacoLinks entry -- an
// app-relative path (e.g. "/run?runID=...") resolves against the current
// origin and is http(s) too. Rejects anything else (javascript:, data:, a
// malformed value) before it can ever reach window.open, since
// monacoLinks is a generic prop on a shared component: today's only
// caller builds hrefs from trusted pathCreator methods, but a future
// caller could populate it from less-trusted data.
function safeNavigationUrl(href: string): string | null {
  let parsed: URL;
  try {
    parsed = new URL(href, window.location.origin);
  } catch {
    return null;
  }
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    return null;
  }
  return parsed.toString();
}

// Computes underline decorations for `links` against ed's own current
// model text and applies them via decorationsCollection.set (replacing
// whatever was there before). Also updates activeLinksRef, which the
// plain mousedown handler (registered once in onMount) reads to resolve a
// click position back to a URL.
//
// Each entry's `text` is located as a literal substring of the model's
// own live text (caller-supplied verbatim -- see CodeBlockProps.
// monacoLinks' own doc comment on quoting). Matches are resolved by
// first-available occurrence with non-overlapping-range tracking
// (`claimed`), not a strictly-advancing cursor -- entries can be given in
// any order (e.g. a {key, id} pair's two entries don't need to know which
// field the underlying JSON serializer happened to emit first).
function applyMonacoLinkDecorations(
  ed: editor.IStandaloneCodeEditor,
  monacoInstance: Monaco,
  links: { text: string; href: string }[] | undefined,
  decorationsCollection: editor.IEditorDecorationsCollection,
  activeLinksRef: { current: ActiveMonacoLink[] }
) {
  const model = ed.getModel();
  if (!model) return;

  if (!links || links.length === 0) {
    decorationsCollection.set([]);
    activeLinksRef.current = [];
    return;
  }

  const modelText = model.getValue();
  const claimed: [number, number][] = [];
  const decorations: editor.IModelDeltaDecoration[] = [];
  const activeLinks: ActiveMonacoLink[] = [];

  for (const { text: needle, href } of links) {
    const url = safeNavigationUrl(href);
    if (!url) continue;

    if (!needle) continue;

    let searchFrom = 0;
    let idx = -1;
    for (;;) {
      idx = modelText.indexOf(needle, searchFrom);
      if (idx === -1) break;
      const end = idx + needle.length;
      if (!claimed.some(([claimedStart, claimedEnd]) => idx < claimedEnd && end > claimedStart)) {
        break;
      }
      searchFrom = idx + 1;
    }
    if (idx === -1) continue;

    const end = idx + needle.length;
    claimed.push([idx, end]);

    const start = model.getPositionAt(idx);
    const stop = model.getPositionAt(end);
    const range = new monacoInstance.Range(
      start.lineNumber,
      start.column,
      stop.lineNumber,
      stop.column
    );
    decorations.push({
      range,
      options: { inlineClassName: 'text-link underline cursor-pointer' },
    });
    activeLinks.push({ range, url });
  }

  decorationsCollection.set(decorations);
  activeLinksRef.current = activeLinks;
}

/**
 * Given pretty-printed JSON text (2-space indent) and a 1-based line number,
 * returns the dot/bracket-notation path for that line (e.g. "data.users[0].name").
 */
function getJsonPathAtLine(text: string, targetLine: number): string | null {
  const lines = text.split('\n');
  if (targetLine < 1 || targetLine > lines.length) return null;

  const pathByDepth: (string | undefined)[] = [];
  const arrayIndices = new Map<number, number>();
  const isArrayDepth = new Set<number>();

  for (let i = 0; i < targetLine; i++) {
    const line = lines[i]!;
    const trimmed = line.trim();
    if (!trimmed) continue;

    const indent = line.length - line.trimStart().length;
    const depth = Math.floor(indent / 2);
    const hasComma = trimmed.endsWith(',');

    // Closing bracket/brace
    if (/^[}\]]/.test(trimmed)) {
      pathByDepth.length = depth;
      if (hasComma && isArrayDepth.has(depth)) {
        arrayIndices.set(depth, (arrayIndices.get(depth) ?? 0) + 1);
      }
      continue;
    }

    // Truncate path to current depth
    pathByDepth.length = depth;

    // If inside an array, set the index segment
    if (isArrayDepth.has(depth)) {
      pathByDepth[depth] = `[${arrayIndices.get(depth) ?? 0}]`;
    }

    // Object property: "key": value
    const keyMatch = trimmed.match(/^"((?:[^"\\]|\\.)*)"\s*:\s*(.*)/);
    if (keyMatch) {
      const key = keyMatch[1]!;
      const rest = keyMatch[2]!.replace(/,?\s*$/, '');
      pathByDepth[depth] = key;
      if (rest === '{') {
        isArrayDepth.delete(depth + 1);
      } else if (rest === '[') {
        isArrayDepth.add(depth + 1);
        arrayIndices.set(depth + 1, 0);
      }
      continue;
    }

    // Standalone { or [
    if (trimmed.startsWith('{')) {
      isArrayDepth.delete(depth + 1);
      continue;
    }
    if (trimmed.startsWith('[')) {
      isArrayDepth.add(depth + 1);
      arrayIndices.set(depth + 1, 0);
      continue;
    }

    // Scalar array element
    if (hasComma && isArrayDepth.has(depth)) {
      arrayIndices.set(depth, (arrayIndices.get(depth) ?? 0) + 1);
    }
  }

  let result = '';
  for (let d = 0; d < pathByDepth.length; d++) {
    const seg = pathByDepth[d];
    if (seg === undefined) continue;
    if (seg.startsWith('[')) {
      result += seg;
    } else {
      result += result ? `.${seg}` : seg;
    }
  }
  return result || null;
}

const jsonParse = (content: string) => {
  try {
    return JSON.parse(content);
  } catch (e) {
    console.error('Error parsing content CodeBlock, returning raw content', e);
    return content;
  }
};

const downloadJson = ({ content }: { content?: string }) => {
  if (content) {
    const blob = new Blob([content], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const element = document.createElement('a');
    element.href = url;
    element.download = 'data.json';
    document.body.appendChild(element);
    element.click();
    document.body.removeChild(element);
    URL.revokeObjectURL(url);
  }
};

export const NewCodeBlock = ({
  header,
  tab,
  actions = [],
  allowFullScreen = false,
  enableTreeView = false,
  loading = false,
  className,
  scrollbarOptions,
  monacoLinks,
}: CodeBlockProps) => {
  const [dark, _] = useState(isDark());
  const [fullScreen, setFullScreen] = useState(false);
  const [mode, setMode] = useState<'tree' | 'raw'>('raw');
  const [wordWrap, setWordWrap] = useLocalStorage('wordWrap', false);
  const { handleCopyClick, isCopying } = useCopyToClipboard();
  // Shared across tree view and raw editor JSON path bars. If both modes are enabled,
  // copying in one then switching can briefly show a stale "Copied" state in the other.
  const { handleCopyClick: handlePathCopyClick, isCopying: isPathCopying } = useCopyToClipboard();
  const [editEmtpy, setEditEmtpy] = useState(false);
  const [hoveredPath, setHoveredPath] = useState<string | null>(null);
  const [cursorPath, setCursorPath] = useState<string | null>(null);
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);
  const monacoInstanceRef = useRef<Monaco | null>(null);
  const linkDecorationsRef = useRef<editor.IEditorDecorationsCollection | null>(null);
  // Read by the plain mousedown handler (registered once in onMount) to
  // resolve a click position back to a URL -- kept up to date by
  // applyMonacoLinkDecorations, called both on mount and from the effect
  // below whenever monacoLinks/content change.
  const activeLinksRef = useRef<ActiveMonacoLink[]>([]);

  const monaco = useMonaco();
  const { content, readOnly = true, language = 'json' } = tab;

  useEffect(() => {
    if (editorRef.current && monacoInstanceRef.current && linkDecorationsRef.current) {
      applyMonacoLinkDecorations(
        editorRef.current,
        monacoInstanceRef.current,
        monacoLinks,
        linkDecorationsRef.current,
        activeLinksRef
      );
    }
    // content is a dependency because monacoLinks' values must be searched
    // for in the model's *current* text -- a content change alone (same
    // monacoLinks reference) still needs decorations recomputed against
    // the new text.
  }, [monacoLinks, content]);

  useEffect(() => {
    if (!monaco) {
      return;
    }

    monaco.editor.defineTheme('inngest-theme', {
      base: dark ? 'vs-dark' : 'vs',
      inherit: true,
      rules: dark ? createRules(true) : createRules(false),
      colors: dark ? createColors(true) : createColors(false),
    });
  }, [monaco, dark]);

  const isOutputTooLarge = (content?.length ?? 0) > maxRenderedOutputSizeBytes;

  if (!monaco) {
    console.info('monaco not loaded, abandoning ship');
    return null;
  }

  return (
    <Fullscreen fullScreen={fullScreen}>
      <div
        className={cn(
          'bg-codeEditor flex h-full flex-col gap-0',
          fullScreen && 'fixed inset-0 z-[52]'
        )}
      >
        <div className={cn('mx-4 mt-2 flex flex-row items-center justify-between gap-4')}>
          <div
            className={cn(
              header?.status === 'error' ? 'text-status-failedText' : 'text-subtle',
              'inline-flex max-h-24 w-0 grow overflow-hidden text-ellipsis whitespace-nowrap text-sm '
            )}
          >
            {enableTreeView ? (
              <SegmentedControl defaultValue={mode}>
                <SegmentedControl.Button value="tree" onClick={() => setMode('tree')}>
                  <div className="overflow-x-hidden overflow-y-hidden text-ellipsis whitespace-nowrap">
                    {'Tree View'}
                  </div>
                </SegmentedControl.Button>
                <SegmentedControl.Button value="raw" onClick={() => setMode('raw')}>
                  <div className="overflow-x-hidden overflow-y-hidden text-ellipsis whitespace-nowrap">
                    {'Raw View'}
                  </div>
                </SegmentedControl.Button>
              </SegmentedControl>
            ) : header?.title ? (
              <Pill
                kind={header?.status === 'error' ? 'error' : 'default'}
                appearance="outlined"
                className="my-2 overflow-x-auto rounded-full p-3"
              >
                <OptionalTooltip
                  tooltip={header.title.length && header.title.length > 55 ? header.title : ''}
                  side="left"
                >
                  <div className="overflow-x-hidden overflow-y-hidden text-ellipsis whitespace-nowrap">
                    {header.title}
                  </div>
                </OptionalTooltip>
              </Pill>
            ) : null}
          </div>

          {!isOutputTooLarge && (
            <div className="flex items-center gap-2">
              {actions.map(({ label, title, icon, onClick, disabled }, idx) => (
                <Button
                  key={idx}
                  icon={icon}
                  onClick={onClick}
                  size="small"
                  aria-label={label}
                  title={title ?? label}
                  label={label}
                  disabled={disabled}
                  appearance="outlined"
                  kind="secondary"
                />
              ))}
              <CopyButton
                size="small"
                code={content}
                isCopying={isCopying}
                handleCopyClick={handleCopyClick}
                appearance="outlined"
              />
              {(!enableTreeView || mode === 'raw') && (
                <Button
                  icon={wordWrap ? <IconOverflowText /> : <IconWrapText />}
                  onClick={() => setWordWrap(!wordWrap)}
                  size="small"
                  aria-label={wordWrap ? 'Do not wrap text' : 'Wrap text'}
                  title={wordWrap ? 'Do not wrap text' : 'Wrap text'}
                  tooltip={wordWrap ? 'Do not wrap text' : 'Wrap text'}
                  appearance="outlined"
                  kind="secondary"
                />
              )}
              {allowFullScreen && (
                <Button
                  onClick={() => setFullScreen(!fullScreen)}
                  size="small"
                  icon={fullScreen ? <RiCollapseDiagonalLine /> : <RiExpandDiagonalLine />}
                  aria-label="Full screen"
                  title="Full screen"
                  tooltip="Full screen"
                  appearance="outlined"
                  kind="secondary"
                />
              )}
            </div>
          )}
        </div>
        <div className={cn('bg-codeEditor flex min-h-0 flex-1 flex-col pt-3')}>
          {isOutputTooLarge && !editEmtpy ? (
            <>
              <Alert severity="warning">Output size is too large to render {`( > 1MB )`}</Alert>
              <div className="bg-codeEditor flex h-24 flex-row items-center justify-center gap-2">
                <Button
                  label="Download Raw"
                  icon={<RiDownload2Line />}
                  onClick={() => downloadJson({ content: content })}
                  appearance="outlined"
                  kind="secondary"
                />
                {!readOnly && (
                  <Button
                    label="Add New Input"
                    icon={<RiEdit2Line />}
                    onClick={() => setEditEmtpy(true)}
                    appearance="outlined"
                    kind="secondary"
                  />
                )}
              </div>
            </>
          ) : loading ? (
            <Skeleton className="h-24 w-full" />
          ) : enableTreeView && mode === 'raw' ? (
            <div className="flex min-h-0 flex-1 flex-col" onMouseLeave={() => setHoveredPath(null)}>
              <div className="min-h-0 flex-1 overflow-y-auto">
                <JSONTree
                  hideRoot={true}
                  data={jsonParse(content) ?? {}}
                  shouldExpandNodeInitially={() => true}
                  theme={jsonTreeTheme(dark)}
                  labelRenderer={(keyPath) => (
                    <span onMouseEnter={() => setHoveredPath(buildJsonPath(keyPath))}>
                      <span className="font-mono text-[13px]">{keyPath[0]}</span>
                      <span className="text-codeDelimiterBracketJson font-mono text-[13px]">:</span>
                    </span>
                  )}
                  valueRenderer={(raw: any) => <span className="font-mono text-[13px]">{raw}</span>}
                  getItemString={() => null}
                />
              </div>
              <div className="bg-canvasSubtle text-muted border-subtle flex min-h-8 shrink-0 items-center justify-between border-t px-3 py-1">
                <code className="truncate font-mono text-xs">
                  {hoveredPath || (
                    <span className="text-muted flex items-center gap-1">
                      <RiInformationLine className="h-3 w-3" />
                      Hover over a JSON key to see full path
                    </span>
                  )}
                </code>
                {hoveredPath && (
                  <CopyButton
                    size="small"
                    code={hoveredPath}
                    isCopying={isPathCopying}
                    handleCopyClick={handlePathCopyClick}
                    appearance="outlined"
                  />
                )}
              </div>
            </div>
          ) : (
            <div className="flex min-h-0 flex-1 flex-col">
              <div className="min-h-0 flex-1">
                <Editor
                  className={cn('h-full', className)}
                  theme="inngest-theme"
                  language={language}
                  value={editEmtpy ? EMPTY_INPUT : content}
                  height="100%"
                  onMount={(ed, monacoInstance) => {
                    editorRef.current = ed;
                    monacoInstanceRef.current = monacoInstance;
                    if (language === 'json') {
                      ed.onDidChangeCursorPosition((e) => {
                        const text = ed.getValue();
                        setCursorPath(getJsonPathAtLine(text, e.position.lineNumber));
                      });
                    }

                    linkDecorationsRef.current = ed.createDecorationsCollection([]);
                    applyMonacoLinkDecorations(
                      ed,
                      monacoInstance,
                      monacoLinks,
                      linkDecorationsRef.current,
                      activeLinksRef
                    );
                    // Plain click (not Monaco's own Cmd/Ctrl+click link
                    // provider, hardcoded into its LinkDetector and not
                    // configurable) -- opens in a new tab when the click
                    // lands inside one of the currently-decorated ranges.
                    // noopener/noreferrer: the opened page must not get a
                    // handle back to this window (reverse tabnabbing).
                    ed.onMouseDown((e) => {
                      if (
                        e.target.type !== monacoInstance.editor.MouseTargetType.CONTENT_TEXT ||
                        !e.target.position
                      ) {
                        return;
                      }
                      const position = e.target.position;
                      const hit = activeLinksRef.current.find(({ range }) =>
                        range.containsPosition(position)
                      );
                      if (hit) {
                        window.open(hit.url, '_blank', 'noopener,noreferrer');
                      }
                    });
                  }}
                  options={{
                    wordWrap: wordWrap ? 'on' : 'off',
                    contextmenu: false,
                    readOnly,
                    minimap: { enabled: false },
                    fontFamily: FONT.font,
                    fontSize: FONT.size,
                    fontWeight: 'light',
                    lineHeight: LINE_HEIGHT,
                    renderLineHighlight: 'none',
                    renderWhitespace: 'none',
                    automaticLayout: true,
                    scrollBeyondLastLine: false,
                    scrollbar: scrollbarOptions ?? {
                      alwaysConsumeMouseWheel: false,
                      horizontalScrollbarSize: 0,
                      verticalScrollbarSize: 0,
                      vertical: 'hidden',
                      horizontal: 'hidden',
                    },
                    guides: {
                      indentation: false,
                      highlightActiveBracketPair: false,
                      highlightActiveIndentation: false,
                    },
                  }}
                />
              </div>
              {language === 'json' && (
                <div className="bg-canvasSubtle text-muted border-subtle flex min-h-8 shrink-0 items-center justify-between border-t px-3 py-1">
                  <code className="truncate font-mono text-xs">
                    {cursorPath || (
                      <span className="text-muted flex items-center gap-1">
                        <RiInformationLine className="h-3 w-3" />
                        Click on a JSON line to see full path
                      </span>
                    )}
                  </code>
                  {cursorPath && (
                    <CopyButton
                      size="small"
                      code={cursorPath}
                      isCopying={isPathCopying}
                      handleCopyClick={handlePathCopyClick}
                      appearance="outlined"
                    />
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </Fullscreen>
  );
};
