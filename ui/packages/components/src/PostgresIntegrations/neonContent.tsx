import { RiCodeBlock, RiLinksFill, RiLockUnlockLine } from '@remixicon/react';

import { IconNeon } from '../icons/platforms/Neon';
import {
  type ConnectPostgresIntegrationContent,
  type PostgresIntegrationMenuContent,
} from './types';

export const neonConnectContent: ConnectPostgresIntegrationContent = {
  title: 'Neon',
  logo: <IconNeon className="text-onContrast" size={20} />,
  // TO DO: Update once we have Neon docs in deploy section
  url: 'https://www.inngest.com/docs/',
  description:
    'This integration enables you to host your Inngest functions on the Vercel platform and automatically sync them every time you deploy code.',
  step: {
    authorize: {
      title: 'Authorize',
      description: 'Add your postgres credentials so Inngest can access your database.',
    },
    'format-wal': {
      title: 'Format WAL',
      description: 'Change/confirm your Write-Ahead Logging (WAL) is set to Logical.',
    },
    'connect-db': {
      title: 'Connect Neon to Inngest',
      description: 'Add your postgres credentials so Inngest can access your database.',
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
