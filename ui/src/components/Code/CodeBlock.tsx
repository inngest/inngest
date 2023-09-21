import { useEffect, useState } from 'react';
import Editor, { useMonaco } from '@monaco-editor/react';

import useCopyToClipboard from '@/hooks/useCopyToClipboard';
import classNames from '../../utils/classnames';
import CopyButton from '../Button/CopyButton';

interface CodeBlockProps {
  tabs: {
    label: string;
    content: string;
  }[];
}

export default function CodeBlock({ tabs }: CodeBlockProps) {
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

  return (
    <div className="w-full bg-slate-800/30 border border-slate-700/30 rounded-lg shadow overflow-hidden">
      <div className="bg-slate-800/40 flex justify-between shadow border-b border-slate-700/20">
        <div className="flex -mb-px">
          {tabs.map((tab, i) => (
            <>
              {tabs.length > 1 ? (
                <button
                  className={classNames(
                    i === activeTab
                      ? `border-indigo-400 text-white`
                      : `border-transparent text-slate-400`,
                    `text-xs px-5 py-2.5 border-b block transition-all duration-150 outline-none`,
                  )}
                  onClick={() => handleTabClick(i)}
                  key={i}
                >
                  {tab.label}
                </button>
              ) : (
                <p key={i} className="text-xs px-5 py-2.5 text-slate-400">
                  {tab.label}
                </p>
              )}
            </>
          ))}
        </div>
        <div className="flex gap-2 items-center mr-2">
          <CopyButton
            code={tabs[activeTab].content}
            isCopying={isCopying}
            handleCopyClick={handleCopyClick}
          />
        </div>
      </div>
      <div>
        {monaco &&
          tabs.map((tab, i) => (
            <div
              className={classNames(
                i === activeTab ? ` ` : `opacity-0 pointer-events-none`,
                `col-start-1 row-start-1 transition-all duration-150`,
              )}
            >
              <Editor
                defaultLanguage="json"
                value={tabs[i].content}
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
                  wordWrap: 'on',
                  fontFamily: 'Roboto_Mono',
                  fontSize: 13,
                  fontWeight: 'light',
                  lineHeight: 26,
                  renderLineHighlight: 'none', // no line selected borders being shown
                  renderWhitespace: 'none', // no indentation spaces being shown
                  guides: {
                    indentation: false, // no indentation vertical lines being shown
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
                  const numberOfLines = editor.getModel()?.getLineCount();
                  // To do: should calculate with getContentHeight instead of number of lines but for some reason the value is wrong
                  if (numberOfLines && numberOfLines <= 10) {
                    // If there are 10 or fewer lines, set the editor's height to the content height
                    const contentHeight = numberOfLines * 26 + 20;
                    editor.layout({ height: contentHeight, width: 0 });
                  } else {
                    // If there are more than 10 lines, set a fixed height with a scrollbar
                    const fixedHeight = 10 * 26 + 20;
                    editor.layout({ height: fixedHeight, width: 0 });
                  }
                }}
              />
            </div>
          ))}
      </div>
    </div>
  );
}
