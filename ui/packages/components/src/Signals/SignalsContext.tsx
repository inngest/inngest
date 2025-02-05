'use client';

import React, { createContext, useContext, useState } from 'react';

export interface InvokeRunPayload {
  functionSlug: string;
  data: Record<string, unknown>;
  user: Record<string, unknown> | null;
}

export type SignalDefinitions = {
  invokeRun: {
    payload: InvokeRunPayload;
    result: unknown;
  };
};

type SignalHandler<TPayload, TResult> = (payload: TPayload) => Promise<TResult>;

export type SignalHandlers = {
  [K in keyof SignalDefinitions]: SignalHandler<
    SignalDefinitions[K]['payload'],
    SignalDefinitions[K]['result']
  >;
};

const SignalsContext = createContext<SignalHandlers | null>(null);

interface SignalsProviderProps {
  handlers: Partial<SignalHandlers>;
  children: React.ReactNode;
}

export const SignalsProvider = ({ handlers, children }: SignalsProviderProps) => {
  return (
    <SignalsContext.Provider value={handlers as SignalHandlers}>{children}</SignalsContext.Provider>
  );
};

export const useSignals = () => {
  const context = useContext(SignalsContext);
  if (!context) {
    throw new Error('useSignals must be used within a SignalsProvider');
  }
  return context;
};
