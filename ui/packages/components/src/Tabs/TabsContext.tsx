import { createContext } from 'react';

// TODO: Add support for 'default' variant
type TabsVariant = 'dynamic';

export interface TabsContextValue {
  defaultIconBefore?: React.ReactNode;
  onClose?: (value: string) => void;
  variant?: TabsVariant;
}

export const TabsContext = createContext<TabsContextValue>({
  variant: 'dynamic',
});
