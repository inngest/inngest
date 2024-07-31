'use client';

import { Link } from '@inngest/components/Link';

import type { Environment } from '@/utils/environments';
import { pathCreator } from '@/utils/urls';
import { Banner } from './Banner';

export function ArchivedEnvBanner({ env }: { env: Environment }) {
  if (!env.isArchived) {
    return null;
  }

  return (
    <Banner kind="warning">
      <span className="font-semibold">Environment is archived.</span> You may unarchive on the{' '}
      <Link className="inline-flex" href={pathCreator.envs()} internalNavigation showIcon={false}>
        environments page
      </Link>
      .
    </Banner>
  );
}
