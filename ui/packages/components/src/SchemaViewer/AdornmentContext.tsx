'use client';

import { createContext, useContext } from 'react';

import type { ValueNode } from './types';

export type RenderAdornmentFn = (node: ValueNode, typeLabel: string) => React.ReactNode;

const defaultRenderAdornment: RenderAdornmentFn = () => null;

const AdornmentContext = createContext<RenderAdornmentFn>(defaultRenderAdornment);

type AdornmentProviderProps = {
  children: React.ReactNode;
  renderAdornment?: RenderAdornmentFn;
};

export function AdornmentProvider({ children, renderAdornment }: AdornmentProviderProps) {
  return (
    <AdornmentContext.Provider value={renderAdornment ?? defaultRenderAdornment}>
      {children}
    </AdornmentContext.Provider>
  );
}

export function useRenderAdornment(): RenderAdornmentFn {
  const ctx = useContext(AdornmentContext);
  if (!ctx) throw new Error('useRenderAdornment must be used within an AdornmentProvider');

  return useContext(AdornmentContext);
}
