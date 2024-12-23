import { CodeLine } from '@inngest/components/CodeLine';
import { NewLink } from '@inngest/components/Link/Link';
import { type AppKind } from '@inngest/components/types/app';
import { RiExternalLinkLine } from '@remixicon/react';

import { type GetAppsQuery } from '@/store/generated';
import UpdateApp from './UpdateApp';

const getAppCardContent = ({ app }: { app: GetAppsQuery['apps'][number] }) => {
  const appKind: AppKind = !app.connected ? 'error' : app.functionCount > 0 ? 'primary' : 'warning';

  const status = !app.connected
    ? 'Not Synced'
    : app.functionCount === 0
    ? 'No functions found'
    : null;

  const footerHeader = !app.connected
    ? app.error === 'unreachable'
      ? `The Inngest Dev Server can't find your application.`
      : `Error: ${app.error}`
    : app.functionCount === 0
    ? 'There are currently no functions registered at this URL.'
    : `${app.functionCount} functions found`;

  const footerContent = !app.connected ? (
    <>
      <p className="text-subtle pb-4">
        Ensure your full URL is correct, including the port. Inngest automatically scans{' '}
        <span className="text-basis">multiple ports</span> by default.
      </p>
      <UpdateApp app={app} />
    </>
  ) : app.functionCount === 0 ? (
    <>
      <p className="text-subtle pb-4">
        Ensure you have created a function and are exporting it correctly from your serve() command.
      </p>
      <CodeLine code="serve(client, [list_of_fns]);" className="mb-4" />
      <NewLink
        size="small"
        target="_blank"
        href="https://www.inngest.com/docs/functions?ref=dev-app"
        iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
      >
        How to write a function
      </NewLink>
    </>
  ) : (
    <ul className="columns-2">
      {[...app.functions]
        .sort((a, b) => a.name.localeCompare(b.name))
        .map((func) => {
          return (
            <li key={func.id} className="text-subtle py-1">
              {func.name}
            </li>
          );
        })}
    </ul>
  );

  return { appKind, status, footerHeader, footerContent };
};

export default getAppCardContent;
