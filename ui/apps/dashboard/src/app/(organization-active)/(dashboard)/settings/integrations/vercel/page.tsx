'use client';

import LoadingIcon from '@/icons/LoadingIcon';
import VercelIntegrationForm from './VercelIntegrationForm';
import { useVercelIntegration } from './getVercelIntegration';

export default function VercelIntegrationPage() {
  const { data, fetching, error } = useVercelIntegration();
  if (fetching) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }
  if (error) {
    throw error;
  }
  return <VercelIntegrationForm vercelIntegration={data} />;
}
