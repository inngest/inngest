'use client';

import React, { createContext, useContext } from 'react';

import type { BooleanFlag } from './useBooleanFlag';
import type { CancelRunPayload, CancelRunResult } from './useCancelRun';
import type { CreateDebugSessionPayload, CreateDebugSessionResult } from './useCreateDebugSession';
import type { DebugRunResult, GetDebugRunPayload } from './useGetDebugRun';
import type { DebugSessionResult, GetDebugSessionPayload } from './useGetDebugSession';
import type { GetRunPayload, GetRunResult } from './useGetRun';
import type { GetRunTracePayload, GetRunTraceResult } from './useGetRunTrace';
import type { GetTraceResultPayload, TraceResult } from './useGetTraceResult';
import type { InngestStatus } from './useInngestStatus';
import type { InvokeRunPayload, InvokeRunResult } from './useInvokeRun';
import type { PathCreator } from './usePathCreator';
import type { RerunPayload, RerunResult } from './useRerun';
import type { RerunFromStepPayload, RerunFromStepResult } from './useRerunFromStep';

//
// These can be either different implementations per app (invokeRun, rerunFromStep) or
// one global implementation
export type SharedDefinitions = {
  getRun: (payload: GetRunPayload) => Promise<GetRunResult>;
  getRunTrace: (payload: GetRunTracePayload) => Promise<GetRunTraceResult>;
  getTraceResult: (payload: GetTraceResultPayload) => Promise<TraceResult>;
  invokeRun: (payload: InvokeRunPayload) => Promise<InvokeRunResult>;
  rerunFromStep: (payload: RerunFromStepPayload) => Promise<RerunFromStepResult>;
  rerun: (payload: RerunPayload) => Promise<RerunResult>;
  cancelRun: (payload: CancelRunPayload) => Promise<CancelRunResult>;
  createDebugSession: (payload: CreateDebugSessionPayload) => Promise<CreateDebugSessionResult>;
  getDebugRun: (payload: GetDebugRunPayload) => Promise<DebugRunResult>;
  getDebugSession: (payload: GetDebugSessionPayload) => Promise<DebugSessionResult>;
  booleanFlag: (flag: string, defaultValue?: boolean) => BooleanFlag;
  inngestStatus: InngestStatus | null;
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
