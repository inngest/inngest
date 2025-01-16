'use client';

import { useEffect, useRef, useState } from 'react';
import { CopyButton } from '@inngest/components/CopyButton';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { cn } from '@inngest/components/utils/classNames';
import {
  FONT,
  LINE_HEIGHT,
  createColors,
  createRules,
  shellLanguageTokens,
} from '@inngest/components/utils/monaco';
import Editor, { useMonaco } from '@monaco-editor/react';
import * as Tabs from '@radix-ui/react-tabs';
import { type editor } from 'monaco-editor';

import { isDark } from '../utils/theme';

export type TabsProps = {
  title: string;
  content: string;
  readOnly?: boolean;
  language?: string;
};

type MonacoEditorType = editor.IStandaloneCodeEditor | null;

const CommandBlock = ({ currentTabContent }: { currentTabContent?: TabsProps }) => {
  const [dark, setDark] = useState(isDark());
  const editorRef = useRef<MonacoEditorType>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const monaco = useMonaco();
  const activeTabContent = currentTabContent || { content: '', readOnly: true, language: 'json' };

  useEffect(() => {
    // We don't have a DOM ref until we're rendered, so check for dark theme parent classes then
    if (wrapperRef.current) {
      setDark(isDark(wrapperRef.current));
    }
  });

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

    monaco.languages.register({ id: 'shell' });
    monaco.languages.setMonarchTokensProvider('shell', shellLanguageTokens);
  }, [monaco, dark]);

  const handleEditorDidMount = (editor: MonacoEditorType) => {
    editorRef.current = editor;
    updateEditorHeight();
  };

  const updateEditorHeight = () => {
    const editor = editorRef.current;
    if (editor) {
      const contentHeight = Math.min(1000, editor.getContentHeight());
      wrapperRef.current!.style.height = `${contentHeight}px`;
      editor.layout();
    }
  };

  return (
    <>
      {monaco && (
        <div ref={wrapperRef}>
          <Editor
            defaultLanguage={activeTabContent.language}
            value={activeTabContent.content}
            theme="inngest-theme"
            onMount={handleEditorDidMount}
            onChange={updateEditorHeight}
            options={{
              readOnly: activeTabContent.readOnly || true,
              minimap: {
                enabled: false,
              },
              lineNumbers: 'off',
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
              scrollbar: {
                verticalScrollbarSize: 10,
                alwaysConsumeMouseWheel: false,
                vertical: 'hidden',
                horizontal: 'hidden',
              },
              padding: {
                top: 10,
                bottom: 10,
              },
              wordWrap: 'off',
              wrappingStrategy: 'advanced',
              overviewRulerLanes: 0,
            }}
          />
        </div>
      )}
    </>
  );
};

CommandBlock.Wrapper = ({ children }: React.PropsWithChildren) => {
  return <div className="border-subtle w-full overflow-hidden rounded-md border">{children}</div>;
};

CommandBlock.Header = ({
  children,
  className,
}: React.PropsWithChildren<{ className?: string }>) => {
  return <div className={cn('border-subtle border-b', className)}>{children}</div>;
};

CommandBlock.CopyButton = ({ content }: { content?: string }) => {
  const { handleCopyClick, isCopying } = useCopyToClipboard();
  return (
    <CopyButton
      size="small"
      code={content}
      isCopying={isCopying}
      handleCopyClick={handleCopyClick}
      appearance="outlined"
    />
  );
};

CommandBlock.Tabs = ({
  tabs,
  activeTab,
  setActiveTab,
}: {
  tabs: TabsProps[];
  activeTab: string;
  setActiveTab?: (tab: string) => void;
}) => {
  const isSingleTab = tabs.length === 1;

  return (
    <Tabs.Root
      value={String(activeTab)}
      onValueChange={(value) => {
        if (setActiveTab) {
          setActiveTab(value);
        }
      }}
    >
      <Tabs.List className="flex h-10 items-stretch px-4">
        {tabs.map((tab) => (
          <Tabs.Trigger
            disabled={isSingleTab}
            key={tab.title}
            value={tab.title}
            className={cn(
              'data-[state=inactive]:text-muted text-basis data-[state=active]:border-contrast border-b-2 border-transparent px-3 py-1 text-sm',
              isSingleTab && 'pl-0 font-medium data-[state=active]:border-transparent'
            )}
          >
            {tab.title}
          </Tabs.Trigger>
        ))}
      </Tabs.List>
    </Tabs.Root>
  );
};

export default CommandBlock;
