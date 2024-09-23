import dynamic from 'next/dynamic';
import { NewButton } from '@inngest/components/Button';
import { Header } from '@inngest/components/Header/Header';
import { Link } from '@inngest/components/Link/Link';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiAddLine, RiQuestionLine } from '@remixicon/react';

import { StatusMenu } from '@/components/Apps/StatusMenu';
import { ServerFeatureFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { pathCreator } from '@/utils/urls';
import { Apps } from './Apps';

const NewUser = dynamic(() => import('@/components/Surveys/NewUser'), {
  ssr: false,
});

const AppInfo = () => (
  <Tooltip>
    <TooltipTrigger>
      <RiQuestionLine className="text-subtle h-[18px] w-[18px]" />
    </TooltipTrigger>
    <TooltipContent
      side="right"
      sideOffset={2}
      className="border-muted text-muted text-md mt-6 flex flex-col rounded-lg border p-0"
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
  const isArchived = archived === 'true';

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Apps' }]}
        infoIcon={<AppInfo />}
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
      <div className="bg-canvasBase mx-auto flex h-full w-full max-w-[1200px] flex-col px-6 pt-16">
        <div className="relative flex w-full flex-row justify-start">
          <StatusMenu archived={isArchived} envSlug={envSlug} />
        </div>
        <Apps isArchived={isArchived} />

        <ServerFeatureFlag flag="new-user-survey">
          <NewUser />
        </ServerFeatureFlag>
      </div>
    </>
  );
}
