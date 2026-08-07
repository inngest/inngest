import AppDetailsCard from '@inngest/components/Apps/AppDetailsCard';
import {
  FeatureReadinessDetail,
  FeatureReadinessTooltip,
  featureReadinessTooltipClassName,
} from '@inngest/components/Apps/FeatureReadiness';
import { Link } from '@inngest/components/Link';
import { Pill } from '@inngest/components/Pill';
import { TextClickToCopy } from '@inngest/components/Text';
import { Time } from '@inngest/components/Time';
import { methodTypes, type App } from '@inngest/components/types/app';
import { RiArrowLeftRightLine, RiInfinityLine } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import { SyncStatusPill } from '@/components/Apps/SyncStatusPill';
import { pathCreator } from '@/utils/urls';
import { PlatformSection } from './PlatformSection';

type Props = {
  //
  // Optional because this card is used in the "unattached syncs" page, and
  // unattached syncs are by definition app-less
  app?: App;

  className?: string;
  sync: Sync | null;
  linkToSyncs?: boolean;
  loading?: false;
  workerCounter?: React.ReactNode;
};

type LoadingProps = {
  app?: undefined;
  className?: string;
  sync?: undefined;
  linkToSyncs?: boolean;
  loading: true;
  workerCounter?: React.ReactNode;
};

type Sync = {
  framework?: string | null;
  lastSyncedAt: Date;
  sdkLanguage?: string | null;
  sdkVersion: string | null;
  status: string;
  url: string | null;
  appVersion: string | null;
} & React.ComponentProps<typeof PlatformSection>['sync'];

function featureReadinessDetail(
  kind: 'aiMetadata' | 'extendedTraces',
  reason?: number | null,
): React.ReactNode {
  if (reason == null) {
    return null;
  }

  return <FeatureReadinessDetail kind={kind} reason={reason} />;
}

function featureReadinessTooltip(
  kind: 'aiMetadata' | 'extendedTraces',
  reason?: number | null,
): React.ReactNode {
  if (reason == null) {
    return null;
  }

  return <FeatureReadinessTooltip kind={kind} />;
}

export const AppInfoCard = ({
  app,
  className,
  sync,
  linkToSyncs,
  loading,
  workerCounter,
}: Props | LoadingProps) => {
  const env = useEnvironment();
  const aiMetadataReason =
    app?.sdkFeatureReadiness?.aiMetadataExtraction?.reason;
  const extendedTracesReason = app?.sdkFeatureReadiness?.extendedTraces?.reason;

  let sdkVersionValue: React.ReactNode = '-';
  if (sync?.sdkVersion) {
    sdkVersionValue = <Pill>{sync.sdkVersion}</Pill>;
  }

  let methodIcon: React.ReactNode = <RiArrowLeftRightLine className="h-4 w-4" />;
  if (app?.method === methodTypes.Connect) {
    methodIcon = <RiInfinityLine className="h-4 w-4" />;
  }

  const appVersion = sync?.appVersion || app?.appVersion;
  let appVersionValue: React.ReactNode = '-';
  if (appVersion) {
    appVersionValue = <Pill>{appVersion}</Pill>;
  }

  let lastSyncValue;
  if (sync) {
    if (app) {
      lastSyncValue = (
        <div className="flex items-center gap-2">
          <SyncStatusPill status={sync.status} />
          {linkToSyncs && <Time value={sync.lastSyncedAt} />}
          {!linkToSyncs && app.externalID && (
            <Link
              to={pathCreator.appSyncs({
                envSlug: env.slug,
                externalAppID: encodeURIComponent(app.externalID),
              })}
              size="small"
            >
              <Time value={sync.lastSyncedAt} />
            </Link>
          )}
        </div>
      );
    } else {
      lastSyncValue = (
        <div className="flex items-center gap-2">
          <SyncStatusPill status={sync.status} />
          <Time value={sync.lastSyncedAt} />
        </div>
      );
    }
  }

  return (
    <>
      <AppDetailsCard title="App information" className={className}>
        <AppDetailsCard.Item
          detail={<div className="truncate">{app?.externalID ?? '-'}</div>}
          term="App ID"
          loading={loading}
        />
        <AppDetailsCard.Item
          detail={
            <div className="truncate">{sdkVersionValue}</div>
          }
          term="SDK version"
          loading={loading}
        />
        <AppDetailsCard.Item
          className="col-span-2"
          detail={<div className="truncate">{lastSyncValue ?? '-'}</div>}
          term="Last sync"
          loading={loading}
        />

        <AppDetailsCard.Item
          detail={<div className="truncate">{sync?.framework ?? '-'}</div>}
          term="Framework"
          loading={loading}
        />
        <AppDetailsCard.Item
          detail={<div className="truncate">{sync?.sdkLanguage || '-'}</div>}
          term="Language"
          loading={loading}
        />
        {aiMetadataReason != null && (
          <AppDetailsCard.Item
            detail={featureReadinessDetail('aiMetadata', aiMetadataReason)}
            term="AI OTel"
            tooltipContent={featureReadinessTooltip(
              'aiMetadata',
              aiMetadataReason,
            )}
            tooltipContentClassName={featureReadinessTooltipClassName}
            tooltipHasArrow={false}
          />
        )}
        {extendedTracesReason != null && (
          <AppDetailsCard.Item
            detail={featureReadinessDetail(
              'extendedTraces',
              extendedTracesReason,
            )}
            term="Extended traces"
            tooltipContent={featureReadinessTooltip(
              'extendedTraces',
              extendedTracesReason,
            )}
            tooltipContentClassName={featureReadinessTooltipClassName}
            tooltipHasArrow={false}
          />
        )}
        <AppDetailsCard.Item
          className="col-span-2"
          detail={
            <TextClickToCopy truncate>{sync?.url ?? '-'}</TextClickToCopy>
          }
          term="URL"
          loading={loading}
        />
        {app?.method && (
          <AppDetailsCard.Item
            term="Method"
            detail={
              <div className="flex items-center gap-1">
                {methodIcon}
                <div className="lowercase first-letter:capitalize">
                  {app.method}
                </div>
              </div>
            }
          />
        )}
        <AppDetailsCard.Item
          detail={
            <div className="truncate">{appVersionValue}</div>
          }
          term="App version"
          loading={loading}
        />
        {app?.method === methodTypes.Connect && workerCounter && (
          <>{workerCounter}</>
        )}

        {sync && <PlatformSection sync={sync} />}
      </AppDetailsCard>
    </>
  );
};
