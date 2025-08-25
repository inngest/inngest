import { createContext, useContext } from 'react';

type Environment = {
  id: string;
  name: string;
  slug: string;
  isArchived: boolean;
  [key: string]: any;
};

type EnvironmentContextType = {
  environment: Environment | null;
  loading: boolean;
  error: string | null;
};

export const EnvironmentContext = createContext<EnvironmentContextType | undefined>(undefined);

export function useEnvironmentContext() {
  const context = useContext(EnvironmentContext);
  if (context === undefined) {
    throw new Error('useEnvironmentContext must be used within an EnvironmentProvider');
  }
  return context;
}

export function EnvironmentProvider({
  children,
  environment,
  loading,
  error,
}: {
  children: React.ReactNode;
  environment: Environment | null;
  loading: boolean;
  error: string | null;
}) {
  return (
    <EnvironmentContext.Provider value={{ environment, loading, error }}>
      {children}
    </EnvironmentContext.Provider>
  );
}
