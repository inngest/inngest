import { deleteConn } from '@/queries/server/integrations/db';
import IntegrationsPage from '@inngest/components/PostgresIntegrations/IntegrationPage';
import { neonIntegrationPageContent } from '@inngest/components/PostgresIntegrations/Neon/newNeonContent';
import type { Publication } from '@inngest/components/PostgresIntegrations/types';

export default function Manage({ publication }: { publication: Publication }) {
  const handleDelete = async (id: string) => {
    try {
      await deleteConn({ data: { id } });
      return { success: true, error: null };
    } catch (error) {
      console.error('Error deleting cdc connection:', error);
      return {
        success: false,
        error: 'Error removing Neon integration, please try again later.',
      };
    }
  };

  return (
    <IntegrationsPage
      publications={[publication]}
      content={neonIntegrationPageContent}
      onDelete={handleDelete}
    />
  );
}
