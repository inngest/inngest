import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';

import { VersionSelect } from '@/components/VersionSelect';

export function baseOptions(): BaseLayoutProps {
  return {
    nav: {
      title: 'Inngest API Docs',
    },
    links: [],
  };
}
