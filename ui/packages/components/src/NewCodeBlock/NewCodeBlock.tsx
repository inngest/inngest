'use client';

import { useEffect, useState, type ReactNode } from 'react';
import { Button } from '@inngest/components/Button';
import { CopyButton } from '@inngest/components/CopyButton';
import { maxRenderedOutputSizeBytes } from '@inngest/components/constants';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { IconOverflowText } from '@inngest/components/icons/OverflowText';
import { IconWrapText } from '@inngest/components/icons/WrapText';
import { cn } from '@inngest/components/utils/classNames';
import { FONT, LINE_HEIGHT, createColors, createRules } from '@inngest/components/utils/monaco';
import Editor, { useMonaco } from '@monaco-editor/react';
import {
  RiCollapseDiagonalLine,
  RiDownload2Line,
  RiEdit2Line,
  RiExpandDiagonalLine,
} from '@remixicon/react';
import { JSONTree } from 'react-json-tree';
import useLocalStorage from 'react-use/lib/useLocalStorage';

import { Alert } from '../Alert';
import { Fullscreen } from '../Fullscreen/Fullscreen';
import { Pill } from '../Pill';
import SegmentedControl from '../SegmentedControl/SegmentedControl';
import { Skeleton } from '../Skeleton';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';
import { jsonTreeTheme } from '../utils/jsonTree';
import { isDark } from '../utils/theme';

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
  parsed?: boolean;
  loading?: boolean;
}

export const NewCodeBlock = ({
  header,
  tab,
  actions = [],
  allowFullScreen = false,
  parsed = true,
  loading = false,
  className,
}: CodeBlockProps) => {
  const [dark, _] = useState(isDark());
  const [fullScreen, setFullScreen] = useState(false);
  const [mode, setMode] = useState<'rich' | 'raw'>('rich');
  const [wordWrap, setWordWrap] = useLocalStorage('wordWrap', false);
  const { handleCopyClick, isCopying } = useCopyToClipboard();
  const [editEmtpy, setEditEmtpy] = useState(false);

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
        <div
          className={cn('bg-codeEditor mx-4 mt-2 flex flex-row items-center justify-between gap-4')}
        >
          <div
            className={cn(
              header?.status === 'error' ? 'text-status-failedText' : 'text-subtle',
              'inline-flex max-h-24 w-0 grow overflow-hidden text-ellipsis whitespace-nowrap text-sm'
            )}
          >
            {parsed ? (
              <SegmentedControl defaultValue={mode}>
                <SegmentedControl.Button value="rich" onClick={() => setMode('rich')}>
                  <div className="overflow-x-hidden overflow-y-hidden text-ellipsis whitespace-nowrap">
                    Parsed {header?.title}
                  </div>
                </SegmentedControl.Button>
                <SegmentedControl.Button value="raw" onClick={() => setMode('raw')}>
                  <div className="w-0 overflow-x-hidden overflow-y-hidden text-ellipsis whitespace-nowrap">
                    Raw {header?.title}
                  </div>
                </SegmentedControl.Button>
              </SegmentedControl>
            ) : (
              <Pill
                kind={header?.status === 'error' ? 'error' : 'default'}
                appearance="outlined"
                className="my-2 overflow-x-auto rounded-full p-3"
              >
                <OptionalTooltip
                  tooltip={header?.title?.length && header?.title?.length > 55 ? header?.title : ''}
                  side="left"
                >
                  <div className="overflow-x-hidden overflow-y-hidden text-ellipsis whitespace-nowrap">
                    {header?.title}
                  </div>
                </OptionalTooltip>
              </Pill>
            )}
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
        <div className={cn('bg-codeEditor h-full overflow-y-auto py-3')}>
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
              className={cn('h-full', className)}
              theme="inngest-theme"
              defaultLanguage={language}
              value={editEmtpy ? EMPTY_INPUT : content}
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
                scrollbar: {
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
          )}
        </div>
      </div>
    </Fullscreen>
  );
};
