'use client';

import type { Environment } from '@/utils/environments';
import { pathCreator } from '@/utils/urls';
import { Banner } from './Banner';

export function ArchivedEnvBanner({ env }: { env: Environment }) {
  if (!env.isArchived) {
    return null;
  }

  return (
    <Banner severity="warning">
      <span className="font-semibold">Environment is archived.</span> You may unarchive on the{' '}
      <Banner.Link severity="warning" className="inline-flex" href={pathCreator.envs()}>
        environments page
      </Banner.Link>
      .
    </Banner>
  );
}
