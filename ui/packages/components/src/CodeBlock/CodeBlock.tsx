'use client';

import { useEffect, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { CopyButton } from '@inngest/components/CopyButton';
import { maxRenderedOutputSizeBytes } from '@inngest/components/constants';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { IconArrayDownTray } from '@inngest/components/icons/ArrayDownTray';
import { IconExpandText } from '@inngest/components/icons/ExpandText';
import { IconOverflowText } from '@inngest/components/icons/OverflowText';
import { IconShrinkText } from '@inngest/components/icons/ShrinkText';
import { IconWrapText } from '@inngest/components/icons/WrapText';
import { classNames } from '@inngest/components/utils/classNames';
import Editor, { useMonaco } from '@monaco-editor/react';
import { type editor } from 'monaco-editor';
import { useLocalStorage } from 'react-use';
import colors from 'tailwindcss/colors';

const LINE_HEIGHT = 26;
const MAX_HEIGHT = 280; // Equivalent to 10 lines + padding
const MAX_LINES = 10;
const FONT = {
  size: 13,
  type: 'monospace',
  font: 'RobotoMono',
};

type MonacoEditorType = editor.IStandaloneCodeEditor | null;

interface CodeBlockProps {
  header?: {
    title?: string;
    description?: string;
    color?: string;
  };
  tabs: {
    label?: string;
    content: string;
    readOnly?: boolean;
    language?: string;
    handleChange?: (value: string) => void;
  }[];
}

export function CodeBlock({ header, tabs }: CodeBlockProps) {
  const [activeTab, setActiveTab] = useState(0);
  const editorRef = useRef<MonacoEditorType>(null);

  const [isWordWrap, setIsWordWrap] = useLocalStorage('isWordWrap', false);
  const [isFullHeight, setIsFullHeight] = useLocalStorage('isFullHeight', false);

  const { handleCopyClick, isCopying } = useCopyToClipboard();

  const monaco = useMonaco();
  const content = tabs[activeTab]?.content;
  const readOnly = tabs[activeTab]?.readOnly ?? true;
  const language = tabs[activeTab]?.language ?? 'json';
  const handleChange = tabs[activeTab]?.handleChange ?? undefined;

  useEffect(() => {
    if (!monaco) {
      return;
    }

    monaco.editor.defineTheme('inngest-theme', {
      base: 'vs-dark',
      inherit: true,
      rules: [
        {
          token: 'delimiter.bracket.json',
          foreground: colors.slate['300'],
        },
        {
          token: 'string.key.json',
          foreground: colors.indigo['400'],
        },
        {
          token: 'number.json',
          foreground: colors.amber['400'],
        },
        {
          token: 'string.value.json',
          foreground: colors.emerald['300'],
        },
        {
          token: 'keyword.json',
          foreground: colors.fuchsia['300'],
        },
        {
          token: 'comment',
          fontStyle: 'italic',
          foreground: colors.slate['500'],
        },
        {
          token: 'string',
          foreground: colors.teal['400'],
        },
        {
          token: 'keyword',
          foreground: colors.indigo['400'],
        },
        {
          token: 'entity.name.function',
          foreground: colors.red['500'],
        },
      ],
      colors: {
        'editor.background': '#1e293b4d', // slate-800/40
        'editorLineNumber.foreground': '#cbd5e14d', // slate-300/30
        'editorLineNumber.activeForeground': colors.slate['300'], // slate-300
        'editorWidget.background': colors.slate['800'],
        'editorWidget.border': colors.slate['500'],
      },
    });
  }, [monaco]);

  useEffect(() => {
    if (editorRef.current) {
      updateEditorLayout(editorRef.current);
    }
  }, [isWordWrap, isFullHeight]);

  const handleTabClick = (index: number) => {
    setActiveTab(index);
  };

  function handleEditorDidMount(editor: MonacoEditorType) {
    editorRef.current = editor;

    const element = document.querySelector('.overflow-guard');
    if (element) {
      element.classList.add('rounded-b-lg');
    }
  }

  function getTextWidth(text: string, font: string) {
    const canvas = document.createElement('canvas');
    const context = canvas.getContext('2d');
    if (context) {
      context.font = font;
      const metrics = context.measureText(text);
      return metrics.width;
    } else {
      return text.length;
    }
  }

  function updateEditorLayout(editor: MonacoEditorType) {
    const container = editor?.getDomNode();
    if (!editor || !container) return;
    const containerWidthWithLineNumbers = container.getBoundingClientRect().width;

    if (!isWordWrap) {
      const contentHeight = editor.getContentHeight();
      const contentHeightWithScroll =
        contentHeight + editor.getLayoutInfo().horizontalScrollbarHeight;

      const linesContent = editor.getModel()?.getLinesContent();
      const containerWidth = containerWidthWithLineNumbers - editor.getLayoutInfo().contentLeft;

      let isScroll = false;

      if (linesContent) {
        for (let lineNumber = 1; lineNumber <= linesContent.length; lineNumber++) {
          const lineContent = linesContent[lineNumber - 1];
          const lineLength = lineContent
            ? getTextWidth(lineContent, `${FONT.size}px ${FONT.font}, ${FONT.type}`)
            : 0;

          if (lineLength > containerWidth) {
            isScroll = true;
            break;
          }
        }
      }

      const newHeight = isScroll ? contentHeightWithScroll : contentHeight;

      if (isFullHeight) {
        editor.layout({ height: newHeight, width: containerWidthWithLineNumbers });
      } else {
        const height = Math.min(MAX_HEIGHT, contentHeight);
        editor.layout({ height: height, width: containerWidthWithLineNumbers });
      }
    }

    if (isWordWrap) {
      const containerWidth =
        container.getBoundingClientRect().width -
        editor.getLayoutInfo().contentLeft -
        editor.getLayoutInfo().verticalScrollbarWidth;
      const linesContent = editor.getModel()?.getLinesContent();
      let totalLinesThatFit = 0;

      if (containerWidth && linesContent && linesContent.length > 0) {
        for (let lineNumber = 1; lineNumber <= linesContent.length; lineNumber++) {
          const lineContent = linesContent[lineNumber - 1];

          const lineLength = lineContent
            ? getTextWidth(lineContent, `${FONT.size}px ${FONT.font}, ${FONT.type}`)
            : 0;

          if (lineLength <= containerWidth) {
            totalLinesThatFit++;
          } else {
            // When using word wrap, monaco breaks keys and values in different lines
            const keyValuePair = lineContent?.split(':');
            let linesNeeded = 1;
            if (keyValuePair && keyValuePair.length === 2 && keyValuePair[0] && keyValuePair[1]) {
              const initialSpaces = (keyValuePair[0]?.match(/^\s*/) || [])[0];
              const keyLength = getTextWidth(
                keyValuePair[0] ?? '',
                `${FONT.size}px ${FONT.font}, ${FONT.type}`
              );
              const valueLength = getTextWidth(
                keyValuePair[1] + initialSpaces,
                `${FONT.size}px ${FONT.font}, ${FONT.type}`
              );
              const keyLinesNeeded = Math.ceil(keyLength / containerWidth);
              const valueLinesNeeded = Math.ceil(valueLength / containerWidth);
              linesNeeded = keyLinesNeeded + valueLinesNeeded;
            } else {
              linesNeeded = Math.ceil(lineLength / containerWidth);
            }
            totalLinesThatFit += linesNeeded;
          }
        }
      }

      if (totalLinesThatFit > MAX_LINES && !isFullHeight) {
        editor.layout({ height: MAX_HEIGHT, width: containerWidthWithLineNumbers });
      } else {
        editor.layout({
          height: totalLinesThatFit * LINE_HEIGHT + 20,
          width: containerWidthWithLineNumbers,
        });
      }
    }
  }

  const handleFullHeight = () => {
    if (editorRef.current) {
      setIsFullHeight(!isFullHeight);
    }
  };

  const handleWrapText = () => {
    const newWordWrap = isWordWrap ? 'off' : 'on';
    setIsWordWrap(!isWordWrap);
    if (editorRef.current) {
      editorRef.current.updateOptions({ wordWrap: newWordWrap });
    }
  };

  // This prevents larger outputs from crashing the browser
  const isOutputTooLarge = (content?.length ?? 0) > maxRenderedOutputSizeBytes;

  const downloadJson = ({ content }: { content?: string }) => {
    if (content) {
      const blob = new Blob([content], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const element = document.createElement('a');
      element.href = url;
      element.download = 'data.json'; // Set the file name with a .json extension
      document.body.appendChild(element);
      element.click();
      document.body.removeChild(element);
      URL.revokeObjectURL(url);
    }
  };

  return (
    <>
      {monaco && (
        <div className="bg-slate-910 w-full rounded-lg border border-slate-700/30 bg-slate-800/40 shadow">
          {header && (
            <div className={classNames(header.color, 'rounded-t-lg pt-3')}>
              {(header.title || header.description) && (
                <div className="flex flex-col gap-1 px-5 pb-2.5 font-mono text-xs">
                  <p className="text-white">{header.title}</p>
                  <p className="text-white/60">{header.description}</p>
                </div>
              )}
            </div>
          )}
          <div
            className={classNames(
              !header && 'rounded-t-lg',
              'flex justify-between border-b border-slate-700/20 bg-slate-800/40 shadow'
            )}
          >
            <div className="-mb-px flex">
              {tabs.map((tab, i) => {
                const isSingleTab = tabs.length === 1;
                const isActive = i === activeTab && !isSingleTab;

                return (
                  <button
                    key={i}
                    className={classNames(
                      `px-5 py-2.5 text-xs`,
                      isSingleTab
                        ? 'text-slate-400'
                        : 'block border-b outline-none transition-all duration-150',
                      isActive && 'border-indigo-400 text-white',
                      !isActive && !isSingleTab && 'border-transparent text-slate-400'
                    )}
                    onClick={() => handleTabClick(i)}
                  >
                    {tab?.label}
                  </button>
                );
              })}
            </div>
            {!isOutputTooLarge && (
              <div className="mr-2 flex items-center gap-2 py-2">
                <CopyButton
                  size="small"
                  code={content}
                  isCopying={isCopying}
                  handleCopyClick={handleCopyClick}
                />
                <Button
                  icon={isWordWrap ? <IconOverflowText /> : <IconWrapText />}
                  btnAction={handleWrapText}
                  size="small"
                  aria-label={isWordWrap ? 'Do not wrap text' : 'Wrap text'}
                  title={isWordWrap ? 'Do not wrap text' : 'Wrap text'}
                />
                <Button
                  btnAction={handleFullHeight}
                  size="small"
                  icon={isFullHeight ? <IconShrinkText /> : <IconExpandText />}
                  aria-label={isFullHeight ? 'Shrink text' : 'Expand text'}
                  title={isFullHeight ? 'Shrink text' : 'Expand text'}
                />
              </div>
            )}
          </div>
          {isOutputTooLarge ? (
            <>
              <div className="bg-amber-500/40 px-5 py-2.5 text-xs text-white">
                Output size is too large to render {`( > 1MB )`}
              </div>
              <div className="flex h-24 items-center justify-center	">
                <Button
                  label="Download Raw"
                  icon={<IconArrayDownTray />}
                  btnAction={() => downloadJson({ content: content })}
                />
              </div>
            </>
          ) : (
            <Editor
              defaultLanguage={language}
              value={content}
              theme="inngest-theme"
              options={{
                extraEditorClassName: 'rounded-b-lg !w-full',
                readOnly: readOnly,
                minimap: {
                  enabled: false,
                },
                lineNumbers: 'on',
                contextmenu: false,
                scrollBeyondLastLine: false,
                fontFamily: FONT.font,
                fontSize: FONT.size,
                fontWeight: 'light',
                lineHeight: LINE_HEIGHT,
                renderLineHighlight: 'none',
                renderWhitespace: 'none',
                guides: {
                  indentation: false,
                  highlightActiveBracketPair: false,
                  highlightActiveIndentation: false,
                },
                scrollbar: { verticalScrollbarSize: 10, alwaysConsumeMouseWheel: false },
                padding: {
                  top: 10,
                  bottom: 10,
                },
                wordWrap: isWordWrap ? 'on' : 'off',
              }}
              onMount={(editor) => {
                handleEditorDidMount(editor);
                updateEditorLayout(editor);
              }}
              onChange={(value) => {
                if (value !== undefined) {
                  handleChange && handleChange(value);
                  updateEditorLayout(editorRef.current);
                }
              }}
            />
          )}
        </div>
      )}
    </>
  );
}
