import { RiLinksFill, RiLockUnlockLine } from '@remixicon/react';

import { IconSupabase } from '../../icons/platforms/Supabase';
import {
  type ConnectPostgresIntegrationContent,
  type IntegrationPageContent,
  type PostgresIntegrationMenuContent,
} from '../types';

export const connectContent: ConnectPostgresIntegrationContent = {
  title: 'Supabase',
  logo: <IconSupabase className="text-onContrast" size={20} />,
  url: 'https://www.inngest.com/docs/features/events-triggers/supabase?ref=app-supabase-connect',
  description:
    'This integration enables you to receive events and trigger functions from your Supabase Postgres database updates.',
  step: {
    authorize: {
      title: 'Authorize',
      description: 'Add your postgres credentials so Inngest can prepare your database.',
    },
    'connect-db': {
      title: 'Connect Supabase to Inngest',
      description: 'Create the connection from Inngest to Supabase',
    },
    // NOTE: Supabase does not include 'format-wal';  it always has wal level logical.
  } as any,
};

export const menuStepContent: PostgresIntegrationMenuContent = {
  title: 'Supabase set up steps',
  step: {
    authorize: {
      title: 'Authorize',
      description: 'Add auth credentials',
      icon: RiLockUnlockLine,
    },
    'connect-db': {
      title: 'Connect',
      description: 'Connect your DB to Inngest',
      icon: RiLinksFill,
    },
    // NOTE: Supabase does not include 'format-wal';  it always has wal level logical.
  } as any,
};

export const integrationPageContent: IntegrationPageContent = {
  title: 'Supabase',
  logo: <IconSupabase className="text-onContrast" size={24} />,
  url: 'https://www.inngest.com/docs/features/events-triggers/supabase?ref=app-supabase-integration-page',
};
