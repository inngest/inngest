'use client';

import React, { createContext, useContext } from 'react';

import type { Run } from '../RunsPage/types';
import type { BooleanFlag } from './useBooleanFlag';
import type { CancelRunPayload, CancelRunResult } from './useCancelRun';
import type { GetRunPayload, GetRunResult } from './useGetRun';
import type { InvokeRunPayload, InvokeRunResult } from './useInvokeRun';
import type { PathCreator } from './usePathCreator';
import type { RerunPayload, RerunResult } from './useRerun';
import type { RerunFromStepPayload, RerunFromStepResult } from './useRerunFromStep';

//
// These can be either different implementations per app (invokeRun, rerunFromStep) or
// one global implementation
export type SharedDefinitions = {
  getRun: (payload: GetRunPayload) => Promise<GetRunResult>;
  invokeRun: (payload: InvokeRunPayload) => Promise<InvokeRunResult>;
  rerunFromStep: (payload: RerunFromStepPayload) => Promise<RerunFromStepResult>;
  rerun: (payload: RerunPayload) => Promise<RerunResult>;
  cancelRun: (payload: CancelRunPayload) => Promise<CancelRunResult>;
  booleanFlag: (flag: string, defaultValue?: boolean) => BooleanFlag;
  pathCreator: PathCreator;
  cloud: boolean;
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
