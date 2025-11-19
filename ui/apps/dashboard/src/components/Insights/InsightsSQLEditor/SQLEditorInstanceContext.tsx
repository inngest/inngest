'use client';

import { createContext, useContext, useRef, type ReactNode } from 'react';
import type { editor } from 'monaco-editor';

type SQLEditorInstanceContextValue = {
  editorRef: React.MutableRefObject<editor.IStandaloneCodeEditor | null>;
};

const SQLEditorInstanceContext = createContext<SQLEditorInstanceContextValue | null>(null);

type SQLEditorInstanceProviderProps = {
  children: ReactNode;
};

export function SQLEditorInstanceProvider({ children }: SQLEditorInstanceProviderProps) {
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);

  return (
    <SQLEditorInstanceContext.Provider value={{ editorRef }}>
      {children}
    </SQLEditorInstanceContext.Provider>
  );
}

export function useSQLEditorInstance() {
  const context = useContext(SQLEditorInstanceContext);
  if (!context) {
    throw new Error('useSQLEditorInstance must be used within SQLEditorInstanceProvider');
  }
  return context;
}
