'use client';

import { useEffect, useState } from 'react';

type Indicator = 'none' | 'minor' | 'major' | 'critical';
type StatusPageStatusResponse = {
  page: {
    id: string;
    name: string;
    url: string;
    updated_at: string;
  };
  status: {
    description: string;
    indicator: Indicator;
  };
};

type Status = {
  url: string;
  description: string;
  indicator: Indicator;
  indicatorColor: string;
  updated_at: string;
};

// We use hex colors b/c tailwind only includes what is initially rendered
export const indicatorColor: { [K in Indicator]: string } = {
  none: '#22c55e', // green-500
  minor: '#fde047', // yellow-300
  major: '#f97316', // orange-500
  critical: '#dc2626', // red-600
};

const STATUS_PAGE_URL = 'https://status.inngest.com';

const fetchStatus = async (): Promise<StatusPageStatusResponse> => {
  return await fetch('https://inngest.statuspage.io/api/v2/status.json').then((r) => r.json());
};

export function useSystemStatus() {
  const [status, setStatus] = useState<Status>({
    url: STATUS_PAGE_URL,
    description: 'Fetching status...',
    indicator: 'none',
    indicatorColor: indicatorColor.none,
    updated_at: '',
  });
  useEffect(() => {
    (async function () {
      const res = await fetchStatus();
      setStatus({
        ...res.status,
        indicatorColor: indicatorColor[res.status.indicator],
        updated_at: res.page.updated_at,
        url: res.page.url,
      });
    })();
  }, []);
  return status;
}
