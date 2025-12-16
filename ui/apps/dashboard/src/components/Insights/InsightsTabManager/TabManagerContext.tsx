'use client';

import { createContext, useContext } from 'react';

import type { Tab } from '../types';
import type { TabManagerActions } from './InsightsTabManager';

interface TabManagerContextValue {
  actions: TabManagerActions;
  activeTab?: Tab;
}

const TabManagerContext = createContext<TabManagerContextValue | null>(null);

interface TabManagerProviderProps {
  children: React.ReactNode;
  actions: TabManagerActions;
  activeTab?: Tab;
}

export function TabManagerProvider({ children, actions, activeTab }: TabManagerProviderProps) {
  return (
    <TabManagerContext.Provider value={{ actions, activeTab }}>
      {children}
    </TabManagerContext.Provider>
  );
}

export function useTabManagerActions(): { tabManagerActions: TabManagerActions } {
  const context = useContext(TabManagerContext);
  if (!context) {
    throw new Error('useTabManagerActions must be used within a TabManagerProvider');
  }

  return { tabManagerActions: context.actions };
}

export function useActiveTab(): { activeTab: Tab | undefined } {
  const context = useContext(TabManagerContext);
  if (!context) {
    throw new Error('useActiveTab must be used within a TabManagerProvider');
  }
  return { activeTab: context.activeTab };
}
