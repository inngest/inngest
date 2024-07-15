'use client';

import { createContext, useContext } from 'react';

import type { Environment } from '@/utils/environments';

export const EnvironmentContext = createContext<Environment | undefined>(undefined);

type EnvironmentProviderProps = {
  env: Environment;
  children: React.ReactNode;
};

export function EnvironmentProvider({ env, children }: EnvironmentProviderProps) {
  return <EnvironmentContext.Provider value={env}>{children}</EnvironmentContext.Provider>;
}

export function useEnvironment() {
  const context = useContext(EnvironmentContext);

  if (!context) {
    throw new Error('useEnvironmentContext must be used inside the EnvironmentProvider.');
  }

  return context;
}
