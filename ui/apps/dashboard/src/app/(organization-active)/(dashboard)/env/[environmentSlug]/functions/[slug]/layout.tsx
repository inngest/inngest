'use client';

import { CodeBracketSquareIcon } from '@heroicons/react/20/solid';
import { Badge } from '@inngest/components/Badge';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import Header, { type HeaderLink } from '@/components/Header/Header';
import { Tag } from '@/components/Tag/Tag';
import { useFunction } from '@/queries';
import ArchiveFunctionButton from './ArchiveButton';
import { InvokeButton } from './InvokeButton';
import PauseFunctionButton from './PauseButton';

type FunctionLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
    slug: string;
  };
};

export default function FunctionLayout({ children, params }: FunctionLayoutProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const [{ data, fetching }] = useFunction({
    functionSlug,
  });

  const fn = data?.workspace.workflow;
  const { isArchived = false } = fn ?? {};
  const isPaused = !isArchived && !data?.workspace.workflow?.current;

  const isReplayEnabled = useBooleanFlag('function-replay');

  const emptyData = !data || fetching || !fn;
  const navLinks: HeaderLink[] = [
    {
      href: `/env/${params.environmentSlug}/functions/${params.slug}`,
      text: 'Dashboard',
      active: 'exact',
    },
    {
      href: `/env/${params.environmentSlug}/functions/${params.slug}/logs`,
      text: 'Runs',
    },
  ];

  if (isReplayEnabled.value) {
    navLinks.push({
      href: `/env/${params.environmentSlug}/functions/${params.slug}/replay`,
      text: 'Replay',
      badge: (
        <Badge kind="solid" className=" h-3.5 bg-indigo-500 px-[0.235rem] text-white">
          New
        </Badge>
      ),
    });
  }

  const doesFunctionAcceptPayload =
    fn?.current?.triggers.some((trigger) => {
      return trigger.eventName;
    }) ?? false;

  return (
    <>
      <Header
        icon={<CodeBracketSquareIcon className="h-3.5 w-3.5 text-white" />}
        title={!data || fetching ? '...' : fn?.name || functionSlug}
        links={navLinks}
        action={
          !emptyData && (
            <div className="flex items-center gap-2">
              {/* Disable buttons that do not yet work */}
              <div className="flex items-center gap-2 pr-2">
                <InvokeButton
                  functionSlug={functionSlug}
                  disabled={isArchived}
                  doesFunctionAcceptPayload={doesFunctionAcceptPayload}
                />
                <PauseFunctionButton functionSlug={functionSlug} disabled={isArchived} />
                <ArchiveFunctionButton functionSlug={functionSlug} />
              </div>
            </div>
          )
        }
        tag={
          !emptyData && isArchived ? (
            <Tag size="sm">Archived</Tag>
          ) : !emptyData && isPaused ? (
            <Tag size="sm" className="text-amber-500">
              Paused
            </Tag>
          ) : null
        }
      />
      {children}
    </>
  );
}
