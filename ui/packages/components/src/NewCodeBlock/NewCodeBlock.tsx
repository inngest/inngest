'use client';

import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Button } from '@inngest/components/Button';
import { CopyButton } from '@inngest/components/CopyButton';
import { maxRenderedOutputSizeBytes } from '@inngest/components/constants';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { IconOverflowText } from '@inngest/components/icons/OverflowText';
import { IconWrapText } from '@inngest/components/icons/WrapText';
import { cn } from '@inngest/components/utils/classNames';
import { FONT, LINE_HEIGHT, createColors, createRules } from '@inngest/components/utils/monaco';
import Editor, { useMonaco } from '@monaco-editor/react';
import { RiCollapseDiagonalLine, RiDownload2Line, RiExpandDiagonalLine } from '@remixicon/react';
import { type editor } from 'monaco-editor';
import { JSONTree } from 'react-json-tree';
import { useLocalStorage } from 'react-use';

import { Alert } from '../Alert';
import { Fullscreen } from '../Fullscreen/Fullscreen';
import SegmentedControl from '../SegmentedControl/SegmentedControl';
import { jsonTreeTheme } from '../utils/jsonTree';
import { isDark } from '../utils/theme';

const DEFAULT_MAX_HEIGHT = 500;

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
  maxHeight?: number;
  allowFullScreen?: boolean;
  parsed?: boolean;
}

export const NewCodeBlock = ({
  header,
  tab,
  actions = [],
  maxHeight = DEFAULT_MAX_HEIGHT,
  allowFullScreen = false,
  parsed = true,
}: CodeBlockProps) => {
  const [dark, _] = useState(isDark());
  const [editorHeight, setEditorHeight] = useState(maxHeight);
  const [fullScreen, setFullScreen] = useState(false);
  const [mode, setMode] = useState<'rich' | 'raw'>('rich');
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [wordWrap, setWordWrap] = useLocalStorage('wordWrap', false);

  const { handleCopyClick, isCopying } = useCopyToClipboard();
  const updateHeight = (editor: editor.IStandaloneCodeEditor) =>
    setEditorHeight(editor.getContentHeight());

  const monaco = useMonaco();
  const { content, readOnly = true, language = 'json' } = tab;

  let parsedContent = null;
  try {
    parsed && (parsedContent = JSON.parse(content));
  } catch (e) {
    console.error('Error parsing content', e);
  }

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

  if (!monaco) {
    console.info('monaco not loaded, abandoning ship');
    return null;
  }

  return (
    <Fullscreen fullScreen={fullScreen}>
      <div
        className={cn(
          'flex h-full flex-col gap-0',
          fullScreen && 'bg-codeEditor fixed inset-0 z-[52]'
        )}
      >
        <div className={cn('bg-canvasSubtle border-subtle min-h-12 border-b')}>
          <div
            className={cn(
              'bg-canvasBase flex items-center justify-between border-l-4 border-l-transparent',
              header?.status === 'error' && 'border-l-status-failed',
              header?.status === 'success' && 'border-l-status-completed'
            )}
          >
            <p
              className={cn(
                header?.status === 'error' ? 'text-status-failedText' : 'text-subtle',
                ' px-5 py-2.5 text-sm',
                'max-h-24 text-ellipsis break-words'
              )}
            >
              {parsed ? (
                <SegmentedControl defaultValue={mode}>
                  <SegmentedControl.Button value="rich" onClick={() => setMode('rich')}>
                    Parsed {header?.title}
                  </SegmentedControl.Button>
                  <SegmentedControl.Button value="raw" onClick={() => setMode('raw')}>
                    Raw {header?.title}
                  </SegmentedControl.Button>
                </SegmentedControl>
              ) : (
                <>{header?.title}</>
              )}
            </p>
            {!isOutputTooLarge && (
              <div className="mr-4 flex items-center gap-2 py-2">
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
        </div>
        <div
          ref={wrapperRef}
          className={`m-0 overflow-y-auto ${
            fullScreen ? `bg-codeEditor` : `bg-canvasBase max-h-[${maxHeight}px]`
          }`}
        >
          {isOutputTooLarge ? (
            <>
              <Alert severity="warning">Output size is too large to render {`( > 1MB )`}</Alert>
              <div className="bg-codeEditor flex h-24 items-center justify-center">
                <Button
                  label="Download Raw"
                  icon={<RiDownload2Line />}
                  onClick={() => downloadJson({ content: content })}
                  appearance="outlined"
                  kind="secondary"
                />
              </div>
            </>
          ) : parsed && mode === 'rich' ? (
            <JSONTree
              hideRoot={true}
              data={parsedContent ?? {}}
              shouldExpandNodeInitially={() => true}
              theme={jsonTreeTheme(dark)}
              labelRenderer={([key]) => (
                <>
                  <span className="font-mono text-[13px]">{key}</span>
                  <span className="text-codeDelimiterBracketJson font-mono text-[13px]">:</span>
                </>
              )}
              valueRenderer={(raw: any) => <span className="font-mono text-[13px]">{raw}</span>}
              getItemString={() => null}
            />
          ) : (
            <Editor
              theme="inngest-theme"
              defaultLanguage={language}
              value={content}
              height={editorHeight}
              options={{
                wordWrap: wordWrap ? 'on' : 'off',
                contextmenu: false,
                readOnly: readOnly,
                minimap: { enabled: false },
                fontFamily: FONT.font,
                fontSize: FONT.size,
                fontWeight: 'light',
                lineHeight: LINE_HEIGHT,
                scrollBeyondLastLine: false,
                scrollbar: {
                  alwaysConsumeMouseWheel: false,
                  horizontal: 'hidden',
                  vertical: 'hidden',
                },
                guides: {
                  indentation: false,
                  highlightActiveBracketPair: false,
                  highlightActiveIndentation: false,
                },
              }}
              onMount={(editor) => editor.onDidContentSizeChange(() => updateHeight(editor))}
            />
          )}
        </div>
      </div>
    </Fullscreen>
  );
};
