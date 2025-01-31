import { Link } from '@inngest/components/Link/Link';
import { type AppKind } from '@inngest/components/types/app';
import { RiExternalLinkLine } from '@remixicon/react';

import { type FlattenedApp } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/useApps';
import { syncKind, syncStatusText } from '@/components/SyncStatusPill';
import { pathCreator } from '@/utils/urls';

const getAppCardContent = ({ app, envSlug }: { app: FlattenedApp; envSlug: string }) => {
  const statusKey = app.status ?? 'default';
  const appKind: AppKind = app.isArchived ? 'default' : syncKind[statusKey] ?? 'default';

  const status = app.isArchived ? 'Archived' : syncStatusText[statusKey] ?? null;

  const footerHeaderTitle = app.error ? (
    `Error: ${app.error}`
  ) : app.functionCount === 0 ? (
    'There are currently no functions registered at this URL.'
  ) : (
    <>
      {app.functionCount} {app.functionCount === 1 ? 'function' : 'functions'} found
    </>
  );

  const footerHeaderSecondaryCTA =
    !app.error && app.functionCount > 0 ? (
      <Link size="small" href={pathCreator.functions({ envSlug: envSlug })} arrowOnHover>
        View functions
      </Link>
    ) : null;

  const footerContent =
    app.functionCount === 0 ? (
      <>
        <p className="text-subtle pb-4">
          Ensure you have created a function and are exporting it correctly from your serve()
          command.
        </p>
        <Link
          size="small"
          target="_blank"
          href="https://www.inngest.com/docs/learn/serving-inngest-functions?ref=cloud-app"
          iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
        >
          How to serve functions
        </Link>
      </>
    ) : (
      <ul className="divide-subtle divide-y">
        {[...app.functions]
          .sort((a, b) => a.name.localeCompare(b.name))
          .map((func) => {
            return (
              <li key={func.id} className="text-subtle py-2">
                {func.name}
              </li>
            );
          })}
      </ul>
    );

  return { appKind, status, footerHeaderTitle, footerHeaderSecondaryCTA, footerContent };
};

export default getAppCardContent;
