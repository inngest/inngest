import { createContext, useContext } from 'react';

interface DIContextType {
  usePathname: () => string;
}

export const DIContext = createContext<DIContextType | null>(null);

export function useDI() {
  const di = useContext(DIContext);
  if (!di) {
    throw new Error('DIContext is not found');
  }
  return di;
}
