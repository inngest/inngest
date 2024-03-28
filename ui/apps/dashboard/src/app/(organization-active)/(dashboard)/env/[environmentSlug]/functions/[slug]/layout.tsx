'use client';

import { useCallback } from 'react';
import { CodeBracketSquareIcon } from '@heroicons/react/20/solid';
import { Badge } from '@inngest/components/Badge';
import { InvokeButton } from '@inngest/components/InvokeButton';
import { useMutation } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import Header, { type HeaderLink } from '@/components/Header/Header';
import { graphql } from '@/gql';
import { useFunction } from '@/queries';
import ArchiveFunctionButton from './ArchiveButton';
import PauseFunctionButton from './PauseButton';

const InvokeFunctionDocument = graphql(`
  mutation InvokeFunction($envID: UUID!, $data: Map, $functionSlug: String!) {
    invokeFunction(envID: $envID, data: $data, functionSlug: $functionSlug)
  }
`);

type FunctionLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
    slug: string;
  };
};

export default function FunctionLayout({ children, params }: FunctionLayoutProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const [{ data, fetching }] = useFunction({ functionSlug });
  const [, invokeFunction] = useMutation(InvokeFunctionDocument);
  const env = useEnvironment();

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

  if (useBooleanFlag('bulk-cancellation-ui').value) {
    navLinks.push({
      href: `/env/${params.environmentSlug}/functions/${params.slug}/cancellation`,
      text: 'Cancellation',
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

  const invokeAction = useCallback(
    (data: Record<string, unknown>) => {
      invokeFunction({
        envID: env.id,
        data,
        functionSlug,
      });
    },
    [env.id, functionSlug, invokeFunction]
  );

  return (
    <>
      <Header
        icon={<CodeBracketSquareIcon className="h-3.5 w-3.5 text-white" />}
        title={!data || fetching ? '...' : fn?.name || functionSlug}
        links={navLinks}
        action={
          !emptyData &&
          !env.isArchived && (
            <div className="flex items-center gap-2">
              {/* Disable buttons that do not yet work */}
              <div className="flex items-center gap-2 pr-2">
                <InvokeButton
                  disabled={isArchived}
                  doesFunctionAcceptPayload={doesFunctionAcceptPayload}
                  btnAction={invokeAction}
                />
                <PauseFunctionButton functionSlug={functionSlug} disabled={isArchived} />
                <ArchiveFunctionButton functionSlug={functionSlug} />
              </div>
            </div>
          )
        }
        badge={
          !emptyData && isArchived ? (
            <Badge kind="solid" className="bg-slate-800 text-slate-400">
              Archived
            </Badge>
          ) : !emptyData && isPaused ? (
            <Badge kind="solid" className="bg-slate-800 text-amber-500">
              Paused
            </Badge>
          ) : null
        }
      />
      {children}
    </>
  );
}
