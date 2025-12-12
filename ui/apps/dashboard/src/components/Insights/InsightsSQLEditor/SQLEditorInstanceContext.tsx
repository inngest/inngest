import { createContext, useContext, useRef, type ReactNode } from 'react';
import type { SQLEditorInstance } from '@inngest/components/SQLEditor/SQLEditor';

type SQLEditorInstanceContextValue = {
  editorRef: React.MutableRefObject<SQLEditorInstance | null>;
};

const SQLEditorInstanceContext =
  createContext<SQLEditorInstanceContextValue | null>(null);

type SQLEditorInstanceProviderProps = {
  children: ReactNode;
};

export function SQLEditorInstanceProvider({
  children,
}: SQLEditorInstanceProviderProps) {
  const editorRef = useRef<SQLEditorInstance | null>(null);

  return (
    <SQLEditorInstanceContext.Provider value={{ editorRef }}>
      {children}
    </SQLEditorInstanceContext.Provider>
  );
}

export function useSQLEditorInstance() {
  const context = useContext(SQLEditorInstanceContext);
  if (!context) {
    throw new Error(
      'useSQLEditorInstance must be used within SQLEditorInstanceProvider',
    );
  }
  return context;
}
