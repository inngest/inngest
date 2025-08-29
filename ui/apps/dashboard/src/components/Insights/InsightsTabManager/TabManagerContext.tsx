'use client';

import { createContext, useContext } from 'react';

import type { TabManagerActions } from './InsightsTabManager';

interface TabManagerContextValue {
  actions: TabManagerActions;
}

const TabManagerContext = createContext<TabManagerContextValue | null>(null);

interface TabManagerProviderProps {
  children: React.ReactNode;
  actions: TabManagerActions;
}

export function TabManagerProvider({ children, actions }: TabManagerProviderProps) {
  return <TabManagerContext.Provider value={{ actions }}>{children}</TabManagerContext.Provider>;
}

export function useTabManagerActions(): { tabManagerActions: TabManagerActions } {
  const context = useContext(TabManagerContext);
  if (!context) {
    throw new Error('useTabManagerActions must be used within a TabManagerProvider');
  }

  return { tabManagerActions: context.actions };
}
