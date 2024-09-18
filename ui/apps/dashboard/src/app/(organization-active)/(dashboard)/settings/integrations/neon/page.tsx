import IntegrationsPage from '@inngest/components/PostgresIntegrations/IntegrationPage';
import { neonIntegrationPageContent } from '@inngest/components/PostgresIntegrations/Neon/neonContent';

const mockedPublications = [
  {
    isActive: true,
    name: 'Publication 1',
  },
];

export default async function Page() {
  return (
    <IntegrationsPage publications={mockedPublications} content={neonIntegrationPageContent} />
  );
}
