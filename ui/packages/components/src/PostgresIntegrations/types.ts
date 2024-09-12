export type IntegrationSteps = 1 | 2 | 3;

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
