'use client';

import { createContext, useContext } from 'react';

import type { ValueNode } from './types';

export type ComputeTypeFn = (node: ValueNode, baseLabel: string) => string;

const defaultComputeType: ComputeTypeFn = (_node, baseLabel) => baseLabel;

const TypeContext = createContext<ComputeTypeFn>(defaultComputeType);

export function TypeProvider({
  children,
  computeType,
}: {
  children: React.ReactNode;
  computeType?: ComputeTypeFn;
}) {
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
