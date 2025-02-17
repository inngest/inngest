'use client';

import React, { createContext, useContext } from 'react';

import type { InvokeRunPayload, InvokeRunResult } from './useInvokeRun';
import type { RerunFromStepPayload, RerunFromStepResult } from './useRerunFromStep';

export type SharedDefinitions = {
  invokeRun: {
    payload: InvokeRunPayload;
    result: InvokeRunResult;
  };
  rerunFromStep: {
    payload: RerunFromStepPayload;
    result: RerunFromStepResult;
  };
};

type SharedHandler<TPayload, TResult> = (payload: TPayload) => Promise<TResult>;

export type SharedHandlers = {
  [K in keyof SharedDefinitions]: SharedHandler<
    SharedDefinitions[K]['payload'],
    SharedDefinitions[K]['result']
  >;
};

const SharedContext = createContext<SharedHandlers | null>(null);

interface SharedProviderProps {
  handlers: Partial<SharedHandlers>;
  children: React.ReactNode;
}

export const SharedProvider = ({ handlers, children }: SharedProviderProps) => {
  return (
    <SharedContext.Provider value={handlers as SharedHandlers}>{children}</SharedContext.Provider>
  );
};

export const useShared = () => {
  const context = useContext(SharedContext);
  if (!context) {
    throw new Error('useShared must be used within a SharedProvider');
  }
  return context;
};
