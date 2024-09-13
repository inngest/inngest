export type IntegrationSteps = 'authorize' | 'format-wal' | 'connect-db';
export const STEPS_ORDER: IntegrationSteps[] = ['authorize', 'format-wal', 'connect-db'];

type ConnectStepContent = {
  title: string;
  description: string;
};

export type ConnectPostgresIntegrationContent = {
  title: string;
  logo: React.ReactNode;
  description: React.ReactNode;
  url?: string;
  step: {
    [K in IntegrationSteps]: ConnectStepContent;
  };
};
