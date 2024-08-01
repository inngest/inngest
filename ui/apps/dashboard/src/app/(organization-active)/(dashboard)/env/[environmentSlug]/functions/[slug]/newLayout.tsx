'use client';

import { useCallback, useState } from 'react';
import { Badge } from '@inngest/components/Badge';
import { InvokeModal } from '@inngest/components/InvokeButton';
import { useMutation } from 'urql';

import { ArchivedAppBanner } from '@/components/ArchivedAppBanner';
import { useEnvironment } from '@/components/Environments/environment-context';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { ActionsMenu } from '@/components/Functions/ActionMenu';
import { CancelFunctionModal } from '@/components/Functions/CancelFunction/CancelFunctionModal';
import { PauseFunctionModal } from '@/components/Functions/PauseFunction/PauseModal';
import { Header, type HeaderLink } from '@/components/Header/Header';
import { graphql } from '@/gql';
import { useFunction } from '@/queries';
import { pathCreator } from '@/utils/urls';
import NewReplayModal from './logs/NewReplayModal';

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

export default function FunctionLayout({
  children,
  params: { environmentSlug, slug },
}: FunctionLayoutProps) {
  const [invokOpen, setInvokeOpen] = useState(false);
  const [pauseOpen, setPauseOpen] = useState(false);
  const [cancelOpen, setCancelOpen] = useState(false);
  const [replayOpen, setReplayOpen] = useState(false);

  const functionSlug = decodeURIComponent(slug);
  const [{ data, fetching }] = useFunction({ functionSlug });
  const [, invokeFunction] = useMutation(InvokeFunctionDocument);
  const env = useEnvironment();

  const fn = data?.workspace.workflow;
  const { isArchived = false, isPaused } = fn ?? {};

  const isNewRunsEnabled = useBooleanFlag('new-runs');
  const isBulkCancellationEnabled = useBooleanFlag('bulk-cancellation-ui');

  const emptyData = !data || fetching || !fn;
  const navLinks: HeaderLink[] = [
    {
      href: `/env/${environmentSlug}/functions/$slug}`,
      text: 'Dashboard',
      active: 'exact',
    },
    {
      href: `/env/${environmentSlug}/functions/${slug}/logs`,
      text: 'Runs',
    },
    {
      href: `/env/${environmentSlug}/functions/${slug}/replay`,
      text: 'Replay',
    },
  ];

  if (isNewRunsEnabled.value) {
    navLinks.push({
      href: `/env/${environmentSlug}/functions/${slug}/runs`,
      text: 'Runs',
      badge: (
        <Badge kind="solid" className=" h-3.5 bg-indigo-500 px-[0.235rem] text-white">
          Beta
        </Badge>
      ),
    });
  }

  if (isBulkCancellationEnabled.value) {
    navLinks.push({
      href: pathCreator.functionCancellations({
        envSlug: environmentSlug,
        functionSlug: slug,
      }),
      text: 'Cancellations',
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
      setInvokeOpen(false);
    },
    [env.id, functionSlug, invokeFunction]
  );

  const externalAppID = data?.workspace.workflow?.appName;

  return (
    <>
      {externalAppID && <ArchivedAppBanner externalAppID={externalAppID} />}
      {invokOpen && (
        <InvokeModal
          doesFunctionAcceptPayload={doesFunctionAcceptPayload}
          isOpen={invokOpen}
          onCancel={() => setInvokeOpen(false)}
          onConfirm={invokeAction}
        />
      )}
      {fn && pauseOpen && (
        <PauseFunctionModal
          functionID={fn.id}
          functionName={fn.name}
          isPaused={fn.isPaused}
          isOpen={pauseOpen}
          onClose={() => setPauseOpen(false)}
        />
      )}
      {cancelOpen && (
        <CancelFunctionModal
          envID={env.id}
          functionSlug={functionSlug}
          isOpen={cancelOpen}
          onClose={() => setCancelOpen(false)}
        />
      )}
      {replayOpen && (
        <NewReplayModal
          isOpen={replayOpen}
          functionSlug={functionSlug}
          onClose={() => setReplayOpen(false)}
        />
      )}
      <Header
        breadcrumb={[
          { text: 'Functions', href: `/env/${environmentSlug}/functions` },
          { text: fn?.name || 'Function', href: `/env/${environmentSlug}/functions/${slug}` },
        ]}
        action={
          <div className="flex flex-row items-center justify-end">
            <ActionsMenu
              showCancel={() => setCancelOpen(true)}
              showInvoke={() => setInvokeOpen(true)}
              showPause={() => setPauseOpen(true)}
              showReplay={() => setReplayOpen(true)}
              archived={isArchived}
              paused={isPaused}
            />
          </div>
        }
      />
      {children}
    </>
  );
}
