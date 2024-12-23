'use client';

import { useMemo } from 'react';
import { AppCard } from '@inngest/components/Apps/AppCard';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { NewLink } from '@inngest/components/Link';
import { Pill } from '@inngest/components/Pill/Pill';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { RiInformationLine } from '@remixicon/react';

import AddAppButton from '@/components/App/AddAppButton';
import AppActions from '@/components/App/AppActions';
import getAppCardContent from '@/components/App/AppCardContent';
import AppFAQ from '@/components/App/AppFAQ';
import { useInfoQuery } from '@/store/devApi';
import { useGetAppsQuery } from '@/store/generated';

export default function AppList() {
  const { data } = useGetAppsQuery(undefined, { pollingInterval: 1500 });
  const apps = data?.apps || [];

  const syncedApps = apps.filter((app) => app.connected === true);
  const numberOfSyncedApps = syncedApps.length;

  const memoizedAppCards = useMemo(() => {
    return apps.map((app) => {
      const { appKind, status, footerHeader, footerContent } = getAppCardContent({ app });

      return (
        <AppCard key={app?.id} kind={appKind}>
          <AppCard.Content
            app={{
              ...app,
              name: !app.name ? 'Syncing...' : app.connected ? `Syncing to ${app.name}` : app.name,
              syncMethod: 'SERVERLESS',
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
                      Autodetected
                    </Pill>
                  )}
                </>
              ) : null
            }
            actions={!app.autodiscovered ? <AppActions id={app.id} name={app.name} /> : null}
          />
          <AppCard.Footer kind={appKind} header={footerHeader} content={footerContent} />
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
              <NewLink
                arrowOnHover
                size="small"
                href="https://www.inngest.com/docs/local-development#connecting-apps-to-the-dev-server"
              >
                Go to specific ports.
              </NewLink>
            }
          />
        }
        action={
          <div className="flex items-center gap-5">
            {info?.isDiscoveryEnabled ? (
              <p className="text-btnPrimary flex items-center gap-2 text-sm leading-tight">
                <IconSpinner className="fill-btnPrimary" />
                Auto-detecting apps
              </p>
            ) : null}
            <AddAppButton />
          </div>
        }
      />

      <div className="mx-auto my-12 w-4/5 max-w-7xl">
        <h2 className="mb-1 text-xl">Synced Apps</h2>
        <p className="text-muted text-sm">
          Synced Inngest apps appear below. Apps will sync automatically if auto-discovery is
          enabled, or you can sync them manually. {''}
          <NewLink
            target="_blank"
            size="small"
            className="inline"
            href="https://www.inngest.com/docs/local-development#connecting-apps-to-the-dev-server"
          >
            Learn more.
          </NewLink>
        </p>
        <div className="bg-surfaceSubtle my-4 mb-4 flex items-center justify-between gap-1 rounded p-4">
          <p className="text-subtle text-sm">
            {numberOfSyncedApps} / {apps.length} apps synced
          </p>
          <div className="flex items-center gap-2">
            {info?.isDiscoveryEnabled ? (
              <p className="text-btnPrimary flex items-center gap-2 text-sm leading-tight">
                <IconSpinner className="fill-btnPrimary" />
                Auto-detecting apps
              </p>
            ) : null}
            <AddAppButton secondary />
          </div>
        </div>
        {info?.isDiscoveryEnabled && (
          <div className="text-light flex items-center gap-1">
            <RiInformationLine className="h-4 w-4" />
            <p className="text-sm">
              Auto-detection is enabled on common ports. You can use the{' '}
              <code className="bg-canvasSubtle text-codeDelimiterBracketJson rounded-sm px-1.5 py-0.5 text-xs">
                --no-discovery
              </code>{' '}
              flag in your CLI to disable it.
            </p>
          </div>
        )}

        <div className="my-6 flex w-full flex-col gap-10">{memoizedAppCards}</div>
        <AppFAQ />
      </div>
    </div>
  );
}
