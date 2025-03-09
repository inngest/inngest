'use client';

import React, { createContext, useContext } from 'react';

import type { InvokeRunPayload, InvokeRunResult } from './useInvokeRun';
import type { LegacyTraceType } from './useLegacyTrace';
import type { RerunPayload, RerunResult } from './useRerun';
import type { RerunFromStepPayload, RerunFromStepResult } from './useRerunFromStep';

//
// These can be either different implementations per app (invokeRun, rerunFromStep) or
// one global implementation (legacyTrace)
export type SharedDefinitions = {
  invokeRun: (payload: InvokeRunPayload) => Promise<InvokeRunResult>;
  rerunFromStep: (payload: RerunFromStepPayload) => Promise<RerunFromStepResult>;
  rerun: (payload: RerunPayload) => Promise<RerunResult>;
  legacyTrace: LegacyTraceType;
};

export type SharedHandlers = SharedDefinitions;

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
