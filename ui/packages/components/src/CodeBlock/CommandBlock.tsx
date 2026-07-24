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
import type { editor } from 'monaco-editor';

import { isDark } from '../utils/theme';

export type TabsProps = {
  title: string;
  content: string;
  readOnly?: boolean;
  language?: string;
  wordWrap?: 'on' | 'off';
};

type MonacoEditorType = editor.IStandaloneCodeEditor | null;

const CommandBlock = ({
  currentTabContent,
  height,
}: {
  currentTabContent?: TabsProps;
  // When set, the editor is pinned to this fixed pixel height (content scrolls
  // internally) instead of growing to fit. Keeps tabbed snippets from jumping.
  height?: number;
}) => {
  const [dark, setDark] = useState(isDark());
  const editorRef = useRef<MonacoEditorType>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const monaco = useMonaco();
  const activeTabContent: TabsProps = currentTabContent || {
    title: '',
    content: '',
    readOnly: true,
    language: 'json',
  };

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

    // These are read-only snippet viewers showing intentionally-incomplete
    // code (free identifiers, fragments). Suppress the TypeScript worker's
    // diagnostics so they don't render error squiggles, and the deprecated-
    // symbol strikethrough (e.g. bare `event` resolving to the deprecated DOM
    // global, which comes from suggestion diagnostics, not semantic tokens).
    // typescriptDefaults is a global singleton, but this app has no TypeScript
    // editing surfaces that need diagnostics (CodeSearch uses `cel`, event
    // editors use `json`), so disabling them globally is safe.
    monaco.languages.typescript.typescriptDefaults.setDiagnosticsOptions({
      noSemanticValidation: true,
      noSyntaxValidation: true,
      noSuggestionDiagnostics: true,
    });
  }, [monaco, dark]);

  const handleEditorDidMount = (editor: MonacoEditorType) => {
    editorRef.current = editor;
    updateEditorHeight();
  };

  const updateEditorHeight = () => {
    const editor = editorRef.current;
    if (editor) {
      const contentHeight = height != null ? height : Math.min(1000, editor.getContentHeight());
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
                // When pinned to a fixed height the editor scrolls internally,
                // so it should consume the wheel rather than leak it to the
                // page. Auto-height blocks don't scroll, so let the page handle
                // the wheel as before.
                alwaysConsumeMouseWheel: height != null,
                vertical: height != null ? 'auto' : 'hidden',
                horizontal: 'hidden',
              },
              padding: {
                top: 10,
                bottom: 10,
              },
              wordWrap: activeTabContent.wordWrap ?? 'off',
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

function CommandBlockCopyButton({
  content,
  onCopy,
}: {
  content?: string;
  // Fired after a successful copy so callers can hook in side effects (e.g.
  // analytics) without this shared component depending on any app.
  onCopy?: () => void;
}) {
  const { handleCopyClick, isCopying } = useCopyToClipboard();
  return (
    <CopyButton
      size="small"
      code={content}
      isCopying={isCopying}
      handleCopyClick={(code) => {
        handleCopyClick(code);
        onCopy?.();
      }}
      appearance="outlined"
    />
  );
}
CommandBlock.CopyButton = CommandBlockCopyButton;

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
