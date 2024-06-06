'use client';

import { Alert } from '@inngest/components/Alert';
import { IconApp } from '@inngest/components/icons/App';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { ArchivedAppBanner } from '@/components/ArchivedAppBanner';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import Header, { type HeaderLink } from '@/components/Header/Header';
import { ArchiveButton } from './ArchiveButton';
import { ResyncButton } from './ResyncButton';
import { ValidateButton } from './ValidateButton';
import { useNavData } from './useNavData';

type Props = React.PropsWithChildren<{
  params: {
    externalID: string;
  };
}>;

export default function Layout({ children, params: { externalID } }: Props) {
  externalID = decodeURIComponent(externalID);
  const env = useEnvironment();
  const isAppValidationEnabled = useBooleanFlag('app-validation');

  const res = useNavData({
    envID: env.id,
    externalAppID: externalID,
  });
  if (res.error) {
    if (res.error.message.includes('no rows')) {
      return (
        <div className="mt-4 flex place-content-center">
          <Alert severity="warning">{externalID} app not found in this environment</Alert>
        </div>
      );
    }
    throw res.error;
  }
  if (res.isLoading && !res.data) {
    return null;
  }

  const navLinks: HeaderLink[] = [
    {
      active: 'exact',
      href: `/env/${env.slug}/apps/${encodeURIComponent(externalID)}`,
      text: 'Info',
    },
    {
      active: 'exact',
      href: `/env/${env.slug}/apps/${encodeURIComponent(externalID)}/syncs`,
      text: 'Syncs',
    },
  ];

  const actions = [];
  if (res.data.latestSync?.url) {
    actions.push(
      <ResyncButton
        appExternalID={externalID}
        disabled={res.data.isArchived}
        platform={res.data.latestSync.platform}
        latestSyncUrl={res.data.latestSync.url}
      />
    );

    if (isAppValidationEnabled.value) {
      actions.push(<ValidateButton latestSyncUrl={res.data.latestSync.url} />);
    }
  }

  actions.push(
    <ArchiveButton
      appID={res.data.id}
      disabled={res.data.isParentArchived}
      isArchived={res.data.isArchived}
    />
  );

  return (
    <>
      {<ArchivedAppBanner externalAppID={externalID} />}
      <Header
        action={<div className="flex gap-4">{actions}</div>}
        icon={<IconApp className="h-5 w-5 text-white" />}
        links={navLinks}
        title={res.data.name}
      />
      <div className="h-full overflow-hidden bg-slate-100">{children}</div>
    </>
  );
}
