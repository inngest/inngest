import { createContext, useContext } from 'react';

import type { TypedNode } from './types';

export type ComputeTypeFn = (node: TypedNode, baseLabel: string) => string;

const defaultComputeType: ComputeTypeFn = (_node, baseLabel) => baseLabel;

const TypeContext = createContext<ComputeTypeFn>(defaultComputeType);

type TypeProviderProps = {
  children: React.ReactNode;
  computeType?: ComputeTypeFn;
};

export function TypeProvider({ children, computeType }: TypeProviderProps) {
  return (
    <TypeContext.Provider value={computeType ?? defaultComputeType}>
      {children}
    </TypeContext.Provider>
  );
}

export function useComputeType(): ComputeTypeFn {
  const ctx = useContext(TypeContext);
  if (!ctx) throw new Error('useComputeType must be used within a TypeProvider');

  return useContext(TypeContext);
}
