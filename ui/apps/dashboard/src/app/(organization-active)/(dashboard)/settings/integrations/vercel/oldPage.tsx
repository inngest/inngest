'use client';

import LoadingIcon from '@/icons/LoadingIcon';
import VercelIntegrationForm from './VercelIntegrationForm';
import { useVercelIntegration } from './useVercelIntegration';

export default function OldVercelIntegrationPage() {
  const { data, fetching, error } = useVercelIntegration();

  if (fetching) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }
  if (error) {
    // HACK - This is a hack as we don't have error codes,
    // match against the error message
    if (!error.message.match(/(haven't integrated with Vercel yet)/gi)) {
      console.log(error);
      throw error;
    }
  }
  return <VercelIntegrationForm vercelIntegration={data} />;
}
