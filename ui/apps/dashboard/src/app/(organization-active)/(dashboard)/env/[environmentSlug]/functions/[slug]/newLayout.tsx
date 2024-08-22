'use client';

import { useCallback, useState } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { InvokeModal } from '@inngest/components/InvokeButton';
import { Pill } from '@inngest/components/Pill';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { RiPauseCircleLine } from '@remixicon/react';
import { useMutation } from 'urql';

import { ArchivedAppBanner } from '@/components/ArchivedAppBanner';
import { useEnvironment } from '@/components/Environments/environment-context';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { ActionsMenu } from '@/components/Functions/ActionMenu';
import { CancelFunctionModal } from '@/components/Functions/CancelFunction/CancelFunctionModal';
import { PauseFunctionModal } from '@/components/Functions/PauseFunction/PauseModal';
import { graphql } from '@/gql';
import { useFunction } from '@/queries';
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

  const isNewRunsEnabled = useBooleanFlag('new-runs');
  const isBulkCancellationEnabled = useBooleanFlag('bulk-cancellation-ui');

  const fn = data?.workspace.workflow;
  const { isArchived = false, isPaused } = fn ?? {};

  const doesFunctionAcceptPayload =
    fn?.current?.triggers.some((trigger) => {
      return trigger.eventName;
    }) ?? false;

  const invokeAction = useCallback(
    ({ data }: { data: Record<string, unknown> }) => {
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
          { text: fn?.name || 'Function' },
        ]}
        infoIcon={
          isPaused && (
            <Pill appearance="solid" kind="warning">
              <RiPauseCircleLine className="mr-1 h-4 w-4" /> Paused
            </Pill>
          )
        }
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
        tabs={[
          {
            children: 'Dashboard',
            href: `/env/${environmentSlug}/functions/${slug}`,
            exactRouteMatch: true,
          },
          { children: 'Runs', href: `/env/${environmentSlug}/functions/${slug}/logs` },
          ...(isNewRunsEnabled.isReady && isNewRunsEnabled.value
            ? [
                {
                  children: (
                    <div className="m-0 flex flex-row items-center justify-start space-x-1 p-0">
                      <div>New runs</div>
                      <Pill appearance="solid" kind="primary">
                        Beta
                      </Pill>
                    </div>
                  ),
                  href: `/env/${environmentSlug}/functions/${slug}/runs`,
                },
              ]
            : []),
          { children: 'Replay history', href: `/env/${environmentSlug}/functions/${slug}/replay` },
          ...(isBulkCancellationEnabled.isReady && isBulkCancellationEnabled.value
            ? [
                {
                  children: 'Cancellation history',
                  href: `/env/${environmentSlug}/functions/${slug}/cancellations`,
                },
              ]
            : []),
        ]}
      />
      {fetching && <Skeleton className="h-36 w-full" />}
      {children}
    </>
  );
}
