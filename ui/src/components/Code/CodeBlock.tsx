import { useEffect, useState } from 'react';
import Editor, { useMonaco } from '@monaco-editor/react';

import useCopyToClipboard from '@/hooks/useCopyToClipboard';
import { IconArrayDownTray } from '@/icons';
import classNames from '@/utils/classnames';
import { maxRenderedOutputSizeBytes } from '@/utils/constants';
import Button from '../Button/Button';
import CopyButton from '../Button/CopyButton';

interface CodeBlockProps {
  header?: {
    title?: string;
    description?: string;
    color?: string;
  };
  tabs: {
    label: string;
    content: string;
  }[];
}

export default function CodeBlock({ header, tabs }: CodeBlockProps) {
  const [activeTab, setActiveTab] = useState(0);
  const { handleCopyClick, isCopying } = useCopyToClipboard();

  const monaco = useMonaco();

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
          foreground: 'cbd5e1', //slate-300
        },
        {
          token: 'string.key.json',
          foreground: '818cf8', //indigo-400
        },
        {
          token: 'number.json',
          foreground: 'fbbf24', //amber-400
        },
        {
          token: 'string.value.json',
          foreground: '6ee7b7', //emerald-300
        },
        {
          token: 'keyword.json',
          foreground: 'f0abfc', //fuschia-300
        },
      ],
      colors: {
        'editor.background': '#1e293b4d', // slate-800/40
        'editorLineNumber.foreground': '#cbd5e14d', // slate-300/30
        'editorLineNumber.activeForeground': '#CBD5E1', // slate-300
      },
    });
  }, [monaco]);

  const handleTabClick = (index) => {
    setActiveTab(index);
  };

  // This prevents larger outputs from crashing the browser
  const isOutputTooLarge = tabs[activeTab].content?.length > maxRenderedOutputSizeBytes;

  const downloadJson = ({ content }) => {
    const blob = new Blob([content], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const element = document.createElement('a');
    element.href = url;
    element.download = 'data.json'; // Set the file name with a .json extension
    document.body.appendChild(element);
    element.click();
    document.body.removeChild(element);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="w-full bg-slate-800/40 border border-slate-700/30 rounded-lg shadow overflow-hidden">
      {header && (
        <div className={classNames(header.color, 'pt-3')}>
          {(header.title || header.description) && (
            <div className="flex flex-col gap-1 font-mono text-2xs px-5 pb-2.5">
              <p className="text-white">{header.title}</p>
              <p className="text-white/60">{header.description}</p>
            </div>
          )}
        </div>
      )}
      <div className="bg-slate-800/40 flex justify-between shadow border-b border-slate-700/20">
        <div className="flex -mb-px">
          {tabs.map((tab, i) => {
            const isSingleTab = tabs.length === 1;
            const isActive = i === activeTab && !isSingleTab;

            return (
              <button
                key={i}
                className={classNames(
                  `text-xs px-5 py-2.5`,
                  isSingleTab
                    ? 'text-slate-400'
                    : 'border-b block transition-all duration-150 outline-none',
                  isActive && 'border-indigo-400 text-white',
                  !isActive && !isSingleTab && 'border-transparent text-slate-400',
                )}
                onClick={() => handleTabClick(i)}
              >
                {tab.label}
              </button>
            );
          })}
        </div>
        {!isOutputTooLarge && (
          <div className="flex gap-2 items-center mr-2">
            <CopyButton
              code={tabs[activeTab].content}
              isCopying={isCopying}
              handleCopyClick={handleCopyClick}
            />
          </div>
        )}
      </div>
      {isOutputTooLarge ? (
        <>
          <div className="px-5 py-2.5 text-3xs bg-amber-500/40 text-white">
            Output size is too large to render {`( > 1MB )`}
          </div>
          <div className="h-24 flex items-center justify-center	">
            <Button
              label="Download Raw"
              icon={<IconArrayDownTray />}
              btnAction={() => downloadJson({ content: tabs[activeTab].content })}
            />
          </div>
        </>
      ) : (
        <div>
          {monaco && (
            <Editor
              defaultLanguage="json"
              value={tabs[activeTab].content}
              theme="inngest-theme"
              options={{
                readOnly: true,
                minimap: {
                  enabled: false,
                },
                lineNumbers: 'on',
                extraEditorClassName: '',
                contextmenu: false,
                scrollBeyondLastLine: false,
                fontFamily: 'Roboto_Mono',
                fontSize: 13,
                fontWeight: 'light',
                lineHeight: 26,
                renderLineHighlight: 'none',
                renderWhitespace: 'none',
                guides: {
                  indentation: false,
                  highlightActiveBracketPair: false,
                  highlightActiveIndentation: false,
                },
                scrollbar: { verticalScrollbarSize: 10 },
                padding: {
                  top: 10,
                  bottom: 10,
                },
              }}
              onMount={(editor) => {
                const contentHeight = editor.getContentHeight();
                if (contentHeight > 295) {
                  editor.layout({ height: 295, width: 0 });
                } else {
                  editor.layout({ height: contentHeight, width: 0 });
                }
              }}
            />
          )}
        </div>
      )}
    </div>
  );
}
