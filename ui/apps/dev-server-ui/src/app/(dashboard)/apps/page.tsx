'use client';

import { useMemo } from 'react';
import { AppCard } from '@inngest/components/Apps/AppCard';
import { Button } from '@inngest/components/Button/Button';
import { InlineCode } from '@inngest/components/Code';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';
import { Pill } from '@inngest/components/Pill/Pill';
import WorkerCounter from '@inngest/components/Workers/ConnectedWorkersDescription';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { transformFramework, transformLanguage } from '@inngest/components/utils/appsParser';
import { RiExternalLinkLine, RiInformationLine } from '@remixicon/react';

import AddAppButton from '@/components/App/AddAppButton';
import AppActions from '@/components/App/AppActions';
import getAppCardContent from '@/components/App/AppCardContent';
import AppFAQ from '@/components/App/AppFAQ';
import { useGetWorkerCount } from '@/hooks/useGetWorkerCount';
import { useInfoQuery } from '@/store/devApi';
import { AppMethod, useGetAppsQuery } from '@/store/generated';

export default function AppList() {
  const { data } = useGetAppsQuery(undefined, { pollingInterval: 1500 });
  const getWorkerCount = useGetWorkerCount();
  const apps = data?.apps || [];

  const syncedApps = apps.filter((app) => app.connected === true);
  const numberOfSyncedApps = syncedApps.length;

  const memoizedAppCards = useMemo(() => {
    return apps.map((app) => {
      const { appKind, status, footerHeaderTitle, footerHeaderSecondaryCTA, footerContent } =
        getAppCardContent({ app });

      return (
        <AppCard key={app?.id} kind={appKind}>
          <AppCard.Content
            app={{
              ...app,
              framework: transformFramework(app.framework),
              sdkLanguage: transformLanguage(app.sdkLanguage),
              url: app.method === AppMethod.Connect ? '' : app.url,
              name: !app.name ? 'Syncing...' : !app.connected ? `Syncing to ${app.name}` : app.name,
            }}
            pill={
              status || app.autodiscovered ? (
                <>
                  {status && (
                    <Pill appearance="outlined" kind={appKind}>
                      {status}
                    </Pill>
                  )}
                  {app.autodiscovered && (
                    <Pill appearance="outlined" kind="default">
                      Auto-detected
                    </Pill>
                  )}
                </>
              ) : null
            }
            actions={
              <div className="items-top flex gap-2">
                {app.method === AppMethod.Connect && (
                  <Button
                    appearance="outlined"
                    label="View details"
                    href={`/apps/app?id=${app.id}`}
                  />
                )}
                {!app.autodiscovered && <AppActions id={app.id} name={app.name} />}
              </div>
            }
            workerCounter={<WorkerCounter appID={app.id} getWorkerCount={getWorkerCount} />}
          />
          <AppCard.Footer
            kind={appKind}
            headerTitle={footerHeaderTitle}
            headerSecondaryCTA={footerHeaderSecondaryCTA}
            content={footerContent}
          />
        </AppCard>
      );
    });
  }, [apps]);

  const { data: info } = useInfoQuery();

  return (
    <div className="flex h-full flex-col overflow-y-scroll">
      <Header
        breadcrumb={[{ text: 'Apps' }]}
        infoIcon={
          <Info
            text="This is a list of all apps. We auto-detect apps that you have defined in specific ports."
            action={
              <Link
                iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
                size="small"
                href="https://www.inngest.com/docs/local-development#connecting-apps-to-the-dev-server"
              >
                Go to specific ports
              </Link>
            }
          />
        }
        action={
          <div className="flex items-center gap-5">
            <AddAppButton />
          </div>
        }
      />

      <div className="mx-auto my-12 w-4/5 max-w-7xl">
        <h2 className="mb-1 text-xl">Synced Apps</h2>
        <p className="text-muted text-sm">
          Apps can be synced manually with the CLI's <InlineCode>-u</InlineCode> flag, a config
          file, the button below, or via auto-discovery.{' '}
          <Link
            target="_blank"
            size="small"
            className="inline"
            href="https://www.inngest.com/docs/dev-server#connecting-apps-to-the-dev-server"
          >
            Learn more
          </Link>
        </p>
        {apps.length === 0 && (
          <>
            <div className="bg-disabled my-4 mb-4 flex items-center justify-between gap-1 rounded p-4">
              <p className="text-subtle text-sm">
                {numberOfSyncedApps} / {apps.length} apps synced
              </p>
              <div className="flex items-center gap-2">
                {info?.isDiscoveryEnabled ? (
                  <p className="text-btnPrimary flex items-center gap-2 text-sm leading-tight">
                    <IconSpinner className="fill-btnPrimary" />
                    Auto-discovering apps
                  </p>
                ) : null}
                <AddAppButton secondary />
              </div>
            </div>
            {info?.isDiscoveryEnabled && (
              <div className="text-light flex items-center gap-1">
                <RiInformationLine className="h-4 w-4" />
                <p className="text-sm">
                  Auto-discovery scans common ports and paths for apps. Use the{' '}
                  <InlineCode>--no-discovery</InlineCode> flag in your CLI to disable it.{' '}
                  <Link
                    target="_blank"
                    size="small"
                    className="inline"
                    href="https://www.inngest.com/docs/dev-server#auto-discovery"
                  >
                    Learn more
                  </Link>
                  .
                </p>
              </div>
            )}
          </>
        )}

        <div className="my-6 flex w-full flex-col gap-10">{memoizedAppCards}</div>
        <AppFAQ />
      </div>
    </div>
  );
}
