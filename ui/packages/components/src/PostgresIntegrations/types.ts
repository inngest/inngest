import { type MenuStepContent } from '../Steps/StepsMenu';

export type IntegrationSteps = 'authorize' | 'format-wal' | 'connect-db';
export const STEPS_ORDER: IntegrationSteps[] = ['authorize', 'format-wal', 'connect-db'];

export function isValidStep(step: string): step is IntegrationSteps {
  return STEPS_ORDER.includes(step as IntegrationSteps);
}

type ConnectStepContent = {
  title: string;
  description: string;
};

export type ConnectPostgresIntegrationContent = {
  title: string;
  logo: React.ReactNode;
  description: React.ReactNode;
  step: {
    [K in IntegrationSteps]: ConnectStepContent;
  };
};

export type PostgresIntegrationMenuContent = {
  title: string;
  step: {
    [K in IntegrationSteps]: MenuStepContent;
  };
};
