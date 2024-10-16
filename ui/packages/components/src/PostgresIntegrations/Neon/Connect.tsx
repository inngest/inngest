import { useState } from 'react';
import { NewLink } from '@inngest/components/Link';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { RiCheckboxCircleFill, RiCloseCircleFill } from '@remixicon/react';

import AutomaticSetup from './ConnectAutomaticSetup';
import ManualSetup from './ConnectManualSetup';

export const StatusIndicator = ({
  loading,
  success,
  error,
}: {
  loading?: boolean;
  success?: boolean;
  error?: boolean;
}) => {
  if (loading)
    return (
      <div className="text-link flex items-center gap-1 text-sm">
        <IconSpinner className="fill-link h-4 w-4" />
        In progress
      </div>
    );
  if (success) return <RiCheckboxCircleFill className="text-success h-4 w-4" />;
  if (error) return <RiCloseCircleFill className="text-error h-5 w-5" />;
};

export type Step =
  | 'logical_replication_enabled'
  | 'user_created'
  | 'roles_granted'
  | 'replication_slot_created'
  | 'publication_created';

export default function Connect({
  onSuccess,
  savedCredentials,
  verifyAutoSetup,
  verifyManualSetup,
  handleLostCredentials,
}: {
  onSuccess: () => void;
  handleLostCredentials: () => void;
  savedCredentials?: string;
  verifyAutoSetup: (variables: {
    adminConn: string;
    engine: string;
    name: string;
    replicaConn?: string;
  }) => Promise<{
    success: boolean;
    error: string;
    steps: {
      [key in Step]: { complete: boolean };
    };
  }>;
  verifyManualSetup: (variables: {
    adminConn: string;
    engine: string;
    name: string;
    replicaConn?: string;
  }) => Promise<{
    success: boolean;
    error: string;
    steps: {
      [key in Step]: { complete: boolean };
    };
  }>;
}) {
  const [isManualSetup, setIsManualSetup] = useState(false);

  return (
    <>
      {isManualSetup ? (
        <ManualSetup
          onSuccess={onSuccess}
          savedCredentials={savedCredentials}
          verifyManualSetup={verifyManualSetup}
          handleLostCredentials={handleLostCredentials}
        />
      ) : (
        <AutomaticSetup
          onSuccess={onSuccess}
          savedCredentials={savedCredentials}
          verifyAutoSetup={verifyAutoSetup}
          handleLostCredentials={handleLostCredentials}
        />
      )}
      <hr className="border-subtle my-6" />
      <div className="flex items-center justify-between text-sm">
        {isManualSetup ? (
          <>
            <p>Want to connect your Neon Database automatically?</p>
            <NewLink size="small" href="" onClick={() => setIsManualSetup(false)}>
              Connect Automatically
            </NewLink>
          </>
        ) : (
          <>
            <p>Want to connect your Neon Database manually?</p>
            <NewLink size="small" href="" onClick={() => setIsManualSetup(true)}>
              Connect Manually
            </NewLink>
          </>
        )}
      </div>
    </>
  );
}
