'use client';

import { createContext, useContext, useState } from 'react';
import { IntegrationSteps } from '@inngest/components/PostgresIntegrations/types';

interface StepsContextType {
  stepsCompleted: IntegrationSteps[];
  setStepsCompleted: (state: IntegrationSteps) => void;
  credentials?: string;
  setCredentials: (state: string) => void;
}

const StepsContext = createContext<StepsContextType | undefined>(undefined);

export function StepsProvider({ children }: React.PropsWithChildren) {
  const [stepsCompleted, setStepsCompleted] = useState<IntegrationSteps[]>([]);
  const [credentials, setCredentials] = useState<string>();

  const addStep = (step: IntegrationSteps) => {
    setStepsCompleted((prevSteps) => {
      if (!prevSteps.includes(step)) {
        return [...prevSteps, step];
      }
      return prevSteps;
    });
  };

  return (
    <StepsContext.Provider
      value={{ stepsCompleted, setStepsCompleted: addStep, credentials, setCredentials }}
    >
      {children}
    </StepsContext.Provider>
  );
}

export function useSteps() {
  const context = useContext(StepsContext);
  if (context === undefined) {
    throw new Error('useSteps must be used within a StepsProvider');
  }
  return context;
}
