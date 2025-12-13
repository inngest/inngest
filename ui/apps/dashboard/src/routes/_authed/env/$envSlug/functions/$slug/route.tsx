import { CancelFunctionModal } from '@/components/Functions/CancelFunction/CancelFunctionModal';
import { PauseFunctionModal } from '@/components/Functions/PauseFunction/PauseModal';
import {
  InvokeFunctionOnboardingDocument,
  FunctionTriggerTypes,
} from '@/gql/graphql';
import { useEnvironment } from '@/components/Environments/environment-context';
import { useFunction } from '@/queries/functions';
import { Header } from '@inngest/components/Header/NewHeader';
import { InvokeModal } from '@inngest/components/InvokeButton';
import { Pill } from '@inngest/components/Pill/NewPill';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { RiPauseCircleLine } from '@remixicon/react';
import { createFileRoute, Outlet } from '@tanstack/react-router';
import { useCallback, useState } from 'react';
import { useMutation } from 'urql';
import { ArchivedAppBanner } from '@/components/Apps/ArchivedAppBanner';
import { ArchivedFuncBanner } from '@/components/Functions/ArchivedFuncBanner';
import NewReplayModal from '@/components/Replay/NewReplayModal';
import { ActionsMenu } from '@/components/Functions/ActionMenu';

export const Route = createFileRoute('/_authed/env/$envSlug/functions/$slug')({
  component: FunctionComponent,
});

function FunctionComponent() {
  const { slug, envSlug } = Route.useParams();
  const [invokOpen, setInvokeOpen] = useState(false);
  const [pauseOpen, setPauseOpen] = useState(false);
  const [cancelOpen, setCancelOpen] = useState(false);
  const [replayOpen, setReplayOpen] = useState(false);

  const functionSlug = decodeURIComponent(slug);
  const [{ data, error, fetching }] = useFunction({ functionSlug });
  const [, invokeFunction] = useMutation(InvokeFunctionOnboardingDocument);
  const env = useEnvironment();

  const isBulkCancellationEnabled = useBooleanFlag('bulk-cancellation-ui');

  const fn = data?.workspace.workflow;
  const { isArchived = false, isPaused } = fn ?? {};

  const doesFunctionAcceptPayload =
    fn?.triggers.some((trigger) => {
      return trigger.type == FunctionTriggerTypes.Event;
    }) ?? false;

  const invokeAction = useCallback(
    ({
      data,
      user,
    }: {
      data: Record<string, unknown>;
      user: Record<string, unknown> | null;
    }) => {
      invokeFunction({
        envID: env.id,
        data,
        user,
        functionSlug,
      });
      setInvokeOpen(false);
    },
    [env.id, functionSlug, invokeFunction],
  );

  const externalAppID = data?.workspace.workflow?.app.name;

  if (error) {
    throw error;
  }

  return (
    <>
      {externalAppID && <ArchivedAppBanner externalAppID={externalAppID} />}
      {fn && <ArchivedFuncBanner funcID={fn.id} />}
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
          { text: 'Functions', href: `/env/${envSlug}/functions` },
          { text: fn?.name || 'Function' },
        ]}
        infoIcon={
          isPaused && (
            <Pill
              kind="warning"
              icon={<RiPauseCircleLine className="h-4 w-4" />}
              iconSide="left"
            >
              Paused
            </Pill>
          )
        }
        loading={fetching}
        action={
          <div className="flex flex-row items-center justify-end gap-2">
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
            href: `/env/${envSlug}/functions/${encodeURIComponent(slug)}`,
            exactRouteMatch: true,
          },
          {
            children: 'Runs',
            href: `/env/${envSlug}/functions/${encodeURIComponent(slug)}/runs`,
          },
          {
            children: 'Replays',
            href: `/env/${envSlug}/functions/${encodeURIComponent(
              slug,
            )}/replays`,
          },
          ...(isBulkCancellationEnabled.isReady &&
          isBulkCancellationEnabled.value
            ? [
                {
                  children: 'Cancellations',
                  href: `/env/${envSlug}/functions/${encodeURIComponent(
                    slug,
                  )}/cancellations`,
                },
              ]
            : []),
        ]}
      />
      <Outlet />
    </>
  );
}
