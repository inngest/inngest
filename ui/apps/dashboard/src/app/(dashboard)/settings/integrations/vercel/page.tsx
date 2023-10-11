import VercelIntegrationForm from './VercelIntegrationForm';
import getVercelIntegration from './getVercelIntegration';

export default async function VercelIntegrationPage() {
  const vercelIntegration = await getVercelIntegration();

  return <VercelIntegrationForm vercelIntegration={vercelIntegration} />;
}
