'use client';

import IntegrationsPage, {
  type Publication,
} from '@inngest/components/PostgresIntegrations/IntegrationPage';
import { neonIntegrationPageContent } from '@inngest/components/PostgresIntegrations/Neon/neonContent';

import { deleteConnection } from '../actions';

export default function Page({ publication }: { publication: Publication }) {
  return (
    <IntegrationsPage
      publication={publication}
      content={neonIntegrationPageContent}
      onDelete={deleteConnection}
    />
  );
}
