import { NewButton } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link/Link';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiAddLine, RiQuestionLine } from '@remixicon/react';

import { StatusMenu } from '@/components/Apps/StatusMenu';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { Header } from '@/components/Header/Header';
import { pathCreator } from '@/utils/urls';
import { Apps } from './Apps';
import Page from './oldPage';

const AppInfo = () => (
  <Tooltip>
    <TooltipTrigger>
      <RiQuestionLine className="text-muted h-[18px] w-[18px]" />
    </TooltipTrigger>
    <TooltipContent
      side="right"
      sideOffset={2}
      className="border-muted text-subtle text-md mt-6 flex flex-col rounded-lg border p-0"
    >
      <div className="border-b px-4 py-2 ">Apps map directly to your products or services.</div>

      <div className="px-4 py-2">
        <Link href={'https://www.inngest.com/docs/apps'} className="text-md">
          Learn how apps work
        </Link>
      </div>
    </TooltipContent>
  </Tooltip>
);

export default async function AppsPage({
  params: { environmentSlug: envSlug },
  searchParams: { archived },
}: {
  params: { environmentSlug: string };
  searchParams: { archived: string };
}) {
  const newIANav = await getBooleanFlag('new-ia-nav');

  if (!newIANav) {
    return <Page />;
  }
  const isArchived = archived === 'true';

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Apps', href: `/env/${envSlug}/apps}` }]}
        icon={<AppInfo />}
        action={
          !isArchived && (
            <NewButton
              kind="primary"
              label="Sync new app"
              href={pathCreator.createApp({ envSlug })}
              icon={<RiAddLine />}
              iconSide="left"
            />
          )
        }
      />
      <div className="bg-canvasBase mx-auto my-16 flex h-full w-full max-w-[1200px] flex-col overflow-y-auto px-6">
        <div className="relative flex w-full flex-row justify-end">
          <StatusMenu archived={isArchived} envSlug={envSlug} />
        </div>
        <Apps isArchived={isArchived} />
      </div>
    </>
  );
}
