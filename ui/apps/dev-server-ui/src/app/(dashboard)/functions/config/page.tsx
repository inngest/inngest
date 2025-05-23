'use client';

import { useSearchParams } from 'next/navigation';

import FunctionConfiguration from '@/app/(dashboard)/functions/config/FunctionConfiguration';

export default async function FunctionDashboardPage() {
  const params = useSearchParams();

  const configuration = {
    cancellations: [
      {
        event: 'inngest/function.failed',
        timeout: '30m',
        condition: null,
      },
    ],
    retries: {
      value: 4,
      isDefault: true,
    },
    priority: '600',
    eventsBatch: null,
    concurrency: [
      {
        scope: 'ACCOUNT',
        limit: {
          value: 2,
          isPlanLimit: true,
        },
        key: '"test"',
      },
      {
        scope: 'ENVIRONMENT',
        limit: {
          value: 3,
          isPlanLimit: true,
        },
        key: '"test-env"',
      },
    ],
    rateLimit: {
      limit: 12,
      period: '66s',
      key: '"event.data.customer_id"',
    },
    debounce: {
      period: '30s',
      key: null,
    },
    throttle: {
      burst: 1,
      key: '"event.data.customer_id"',
      limit: 11,
      period: '1m1s',
    },
  };

  return (
    <div>
      This is a function config page yot1 {params.get('slug')}
      <FunctionConfiguration configuration={configuration} />
    </div>
  );
}
