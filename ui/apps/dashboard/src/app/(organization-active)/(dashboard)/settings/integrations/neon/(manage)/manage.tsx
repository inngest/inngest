'use client';

import IntegrationsPage from '@inngest/components/PostgresIntegrations/IntegrationPage';
import { neonIntegrationPageContent } from '@inngest/components/PostgresIntegrations/Neon/neonContent';
import type { Publication } from '@inngest/components/PostgresIntegrations/types.js';

import { deleteConnection } from '../actions';

export default function Manage({ publication }: { publication: Publication }) {
  return (
    <IntegrationsPage
      publications={[publication]}
      content={neonIntegrationPageContent}
      onDelete={deleteConnection}
    />
  );
}
