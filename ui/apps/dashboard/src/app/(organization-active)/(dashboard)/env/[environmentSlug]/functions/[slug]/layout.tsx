'use client';

import { useCallback } from 'react';
import { Badge } from '@inngest/components/Badge';
import { InvokeButton } from '@inngest/components/InvokeButton';
import { IconFunction } from '@inngest/components/icons/Function';
import { useMutation } from 'urql';

import { ArchivedAppBanner } from '@/components/ArchivedAppBanner';
import { useEnvironment } from '@/components/Environments/environment-context';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import Header, { type HeaderLink } from '@/components/Header/Header';
import { graphql } from '@/gql';
import { useFunction } from '@/queries';
import { CancelFunctionButton } from './CancelFunctionButton';
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
  const { isArchived = false, isPaused } = fn ?? {};

  const isNewRunsEnabled = useBooleanFlag('new-runs');

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
    {
      href: `/env/${params.environmentSlug}/functions/${params.slug}/replay`,
      text: 'Replay',
    },
  ];

  if (isNewRunsEnabled.value) {
    navLinks.push({
      href: `/env/${params.environmentSlug}/functions/${params.slug}/runs`,
      text: 'Runs',
      badge: (
        <Badge kind="solid" className=" h-3.5 bg-indigo-500 px-[0.235rem] text-white">
          Beta
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

  const externalAppID = data?.workspace.workflow?.appName;

  return (
    <>
      {externalAppID && <ArchivedAppBanner externalAppID={externalAppID} />}
      <Header
        icon={<IconFunction className="h-3.5 w-3.5 text-white" />}
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
                <CancelFunctionButton envID={env.id} functionSlug={functionSlug} />
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
