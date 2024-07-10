'use client';

import { Link } from '@inngest/components/Link';

import { useEnvironment } from '@/components/Environments/EnvContext';
import { pathCreator } from '@/utils/urls';
import { Banner } from '../../Banner';

export function ArchivedEnvBanner() {
  const env = useEnvironment();
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
