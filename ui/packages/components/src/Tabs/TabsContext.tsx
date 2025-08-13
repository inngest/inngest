import { createContext } from 'react';

// TODO: Add support for 'default' variant
type TabsVariant = 'dynamic';

export interface TabsContextValue {
  onClose?: (value: string) => void;
  variant?: TabsVariant;
}

export const TabsContext = createContext<TabsContextValue>({
  variant: 'dynamic',
});
