import { createServerFn } from '@tanstack/react-start';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';

export const getSandboxAPIEnabled = createServerFn({
  method: 'GET',
}).handler(() =>
  getBooleanFlag('sandbox_api', {
    defaultValue: false,
  }),
);
