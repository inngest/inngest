'use client';

import { createContext, useContext } from 'react';

import LoadingIcon from '@/icons/LoadingIcon';
import { useEnvironment as useEnvironmentQuery } from '@/queries';
import type { Environment } from '@/utils/environments';

export const EnvironmentContext = createContext<Environment | undefined>(undefined);

type EnvironmentProviderProps = {
  environmentSlug: string;
  children: React.ReactNode;
};

export function EnvironmentProvider({ environmentSlug, children }: EnvironmentProviderProps) {
  const [{ data: environment, fetching, error }] = useEnvironmentQuery(environmentSlug);

  if (error) throw error;

  if (fetching) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  if (!environment) {
    throw new Error('Failed to load environment');
  }

  return <EnvironmentContext.Provider value={environment}>{children}</EnvironmentContext.Provider>;
}

export function useEnvironment() {
  const context = useContext(EnvironmentContext);

  if (!context) {
    throw new Error('useEnvironmentContext must be used inside the EnvironmentProvider.');
  }

  return context;
}
