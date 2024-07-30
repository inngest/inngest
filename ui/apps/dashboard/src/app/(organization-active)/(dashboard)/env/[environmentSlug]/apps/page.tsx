import { NewButton } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link/Link';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiAddLine, RiQuestionLine } from '@remixicon/react';

import { ActiveMenu } from '@/components/Apps/ActiveMenu';
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
      <div className="border-b p-4 ">Apps map directly to your products or services.</div>

      <div className="p-4">
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
        breadcrumb={['Apps']}
        icon={<AppInfo />}
        action={
          <div className="flex items-center gap-2">
            {!isArchived && (
              <NewButton
                kind="primary"
                label="Sync new app"
                href={pathCreator.createApp({ envSlug })}
                icon={<RiAddLine />}
                iconSide="left"
              />
            )}
          </div>
        }
      />
      <div className="bg-canvasBase relative my-16 flex h-full flex-col overflow-y-auto px-6">
        <div className="relative flex flex-row justify-end">
          <ActiveMenu archived={isArchived} envSlug={envSlug} />
        </div>
        <Apps isArchived={isArchived} />
      </div>
    </>
  );
}
