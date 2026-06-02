import { deleteConn } from '@/queries/server/integrations/db';
import IntegrationsPage from '@inngest/components/PostgresIntegrations/IntegrationPage';
import { neonIntegrationPageContent } from '@inngest/components/PostgresIntegrations/Neon/newNeonContent';
import type { Publication } from '@inngest/components/PostgresIntegrations/types';

export default function Manage({
  publications,
}: {
  publications: Publication[];
}) {
  const handleDelete = async (id: string) => {
    try {
      await deleteConn({ data: { id } });
      return { success: true, error: null };
    } catch (error) {
      console.error('Error deleting cdc connection:', error);
      const message = error instanceof Error ? error.message : 'Unknown error';
      return {
        success: false,
        error: `Error removing Neon integration: ${message}. Please try again later.`,
      };
    }
  };

  return (
    <IntegrationsPage
      publications={publications}
      content={neonIntegrationPageContent}
      onDelete={handleDelete}
    />
  );
}
