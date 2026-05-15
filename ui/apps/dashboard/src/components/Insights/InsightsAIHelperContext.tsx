import { createContext, useContext, type ReactNode } from 'react';

export interface InsightsAIHelperContextValue {
  openAIHelperWithPrompt: (prompt: string) => Promise<void>;
}

const InsightsAIHelperContext =
  createContext<InsightsAIHelperContextValue | null>(null);

export function InsightsAIHelperProvider({
  children,
  openAIHelperWithPrompt,
}: {
  children: ReactNode;
  openAIHelperWithPrompt: (prompt: string) => Promise<void>;
}) {
  return (
    <InsightsAIHelperContext.Provider value={{ openAIHelperWithPrompt }}>
      {children}
    </InsightsAIHelperContext.Provider>
  );
}

export function useInsightsAIHelper(): InsightsAIHelperContextValue | null {
  return useContext(InsightsAIHelperContext);
}
