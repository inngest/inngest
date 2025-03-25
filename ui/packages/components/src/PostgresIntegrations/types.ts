import { type MenuStepContent } from '../Steps/StepsMenu';

export enum IntegrationSteps {
  Authorize = 'authorize',
  FormatWal = 'format-wal',
  ConnectDb = 'connect-db',
}

export const STEPS_ORDER: IntegrationSteps[] = [
  IntegrationSteps.Authorize,
  IntegrationSteps.FormatWal,
  IntegrationSteps.ConnectDb,
];

export function isValidStep(step: string): step is IntegrationSteps {
  return STEPS_ORDER.includes(step as IntegrationSteps);
}

type ConnectStepContent = {
  title: string;
  description: string;
  // skip indicates if the current Postgres provider's integration skips
  // this step.
  skip?: boolean;
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

export type PostgresIntegrationMenuContent = {
  title: string;
  step: {
    [K in IntegrationSteps]: MenuStepContent;
  };
};

export type IntegrationPageContent = {
  title: string;
  logo: React.ReactNode;
  url: string;
};

export function parseConnectionString(connectionString: string) {
  const regex = /postgresq?l?:\/\/(\w+):([^@]+)@([^/]+)/;
  const match = connectionString.match(regex);

  if (match) {
    const [, username, password, host] = match;
    return {
      name: `Neon-${host}`,
      engine: 'postgresql',
      adminConn: connectionString,
    };
  }

  return null;
}

export type Publication = {
  id: string;
  name: string;
  slug: string;
  projects: never[];
  enabled: boolean;
};
