import { RiCodeBlock, RiLinksFill, RiLockUnlockLine } from '@remixicon/react';

import { IconNeon } from '../../icons/platforms/Neon';
import {
  type ConnectPostgresIntegrationContent,
  type IntegrationPageContent,
  type PostgresIntegrationMenuContent,
} from '../types';

export const neonConnectContent: ConnectPostgresIntegrationContent = {
  title: 'Neon',
  logo: <IconNeon className="text-onContrast" size={20} />,
  url: 'https://www.inngest.com/docs/features/events-triggers/neon?ref=app-neon-connect',
  description:
    'This integration enables you to trigger Inngest functions from your Neon Postgres database updates.',
  step: {
    authorize: {
      title: 'Authorize',
      description: 'Add your postgres credentials so Inngest can prepare your database.',
    },
    'format-wal': {
      title: 'Format WAL',
      description: 'Change/confirm your Write-Ahead Logging (WAL) is set to Logical.',
    },
    'connect-db': {
      title: 'Connect Neon to Inngest',
      description: 'Create the connection from Inngest to Neon',
    },
  },
};

export const neonMenuStepContent: PostgresIntegrationMenuContent = {
  title: 'Neon set up steps',
  step: {
    authorize: {
      title: 'Authorize',
      description: 'Add auth credentials',
      icon: RiLockUnlockLine,
    },
    'format-wal': {
      title: 'Enable logical replication',
      description: 'Correctly format WAL',
      icon: RiCodeBlock,
    },
    'connect-db': {
      title: 'Connect',
      description: 'Connect your DB to Inngest',
      icon: RiLinksFill,
    },
  },
};

export const neonIntegrationPageContent: IntegrationPageContent = {
  title: 'Neon',
  logo: <IconNeon className="text-onContrast" size={24} />,
  url: 'https://www.inngest.com/docs/features/events-triggers/neon?ref=app-neon-integration-page',
};
