import { useEffect, useState } from 'react';

import { getStatus, type ExtendedStatus } from './Status';

export const useSystemStatus = () => {
  const [status, setStatus] = useState<ExtendedStatus | undefined>();

  useEffect(() => {
    (async () => {
      const newStatus = await getStatus();
      if (newStatus) {
        setStatus(newStatus);
      }
    })();
  }, []);

  return status;
};
