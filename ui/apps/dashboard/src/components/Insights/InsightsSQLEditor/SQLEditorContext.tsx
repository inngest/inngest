import {
  createContext,
  useCallback,
  useContext,
  useRef,
  type ReactNode,
} from 'react';
import { flushSync } from 'react-dom';
import * as Sentry from '@sentry/tanstackstart-react';

import type { SQLEditorInstance } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { formatSQL } from './utils';

type SQLEditorContextValue = {
  editorRef: React.MutableRefObject<SQLEditorInstance | null>;
  setQueryAndRun: (sql: string) => void;
  setQuery: (sql: string) => void;
  formatQuery: () => void;
};

const SQLEditorContext = createContext<SQLEditorContextValue | null>(null);

type SQLEditorProviderProps = {
  children: ReactNode;
};

export function SQLEditorProvider({ children }: SQLEditorProviderProps) {
  const { onChange, runQuery } = useInsightsStateMachineContext();
  const editorRef = useRef<SQLEditorInstance | null>(null);

  const setQuery = useCallback(
    (sql: string) => {
      const editor = editorRef.current;
      if (!editor) {
        // Fallback to state-based update if editor not available
        onChange(sql);
        return;
      }

      // Use Monaco editor API to set the value
      const model = editor.getModel();
      if (model) {
        model.setValue(sql);
      } else {
        onChange(sql);
      }
    },
    [editorRef, onChange],
  );

  const formatQuery = useCallback(() => {
    const editor = editorRef.current;
    if (!editor) return;
    editor.getAction('editor.action.formatDocument')?.run();
  }, [editorRef]);

  const setQueryAndRun = useCallback(
    (sql: string) => {
      // Format the SQL using our custom formatter
      const formattedSql = formatSQL(sql.trim());

      // Set the query in the editor (updates Monaco)
      setQuery(formattedSql);

      // Force synchronous state update before running query
      flushSync(() => {
        onChange(formattedSql);
      });

      // Run the query immediately - state is now guaranteed to be synced
      queueMicrotask(() => {
        try {
          runQuery();
        } catch (error) {
          Sentry.captureException(error);
        }
      });
    },
    [setQuery, onChange, runQuery],
  );

  return (
    <SQLEditorContext.Provider
      value={{ editorRef, setQueryAndRun, setQuery, formatQuery }}
    >
      {children}
    </SQLEditorContext.Provider>
  );
}

export function useSQLEditor() {
  const context = useContext(SQLEditorContext);
  if (!context) {
    throw new Error('useSQLEditor must be used within SQLEditorProvider');
  }
  return context;
}

// Convenience hooks for specific concerns
// Note: This hook is used in QueryActionsMenu which can be rendered outside of SQLEditorProvider
// (e.g., in QueryHelperPanel sidebar), so it returns null when context is unavailable
export function useSQLEditorInstance(): {
  editorRef: React.MutableRefObject<SQLEditorInstance | null>;
} | null {
  const context = useContext(SQLEditorContext);
  if (!context) {
    return null;
  }
  return { editorRef: context.editorRef };
}

export function useSQLEditorActions(): {
  setQueryAndRun: (sql: string) => void;
  setQuery: (sql: string) => void;
  formatQuery: () => void;
} | null {
  const context = useContext(SQLEditorContext);
  if (!context) {
    return null;
  }
  const { setQueryAndRun, setQuery, formatQuery } = context;
  return { setQueryAndRun, setQuery, formatQuery };
}
