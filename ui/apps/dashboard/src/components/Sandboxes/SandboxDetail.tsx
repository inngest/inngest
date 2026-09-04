import { CopyButton } from '@inngest/components/CopyButton';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { Header } from '@inngest/components/Header/Header';
import { StatusCell } from '@inngest/components/Table/Cell';
import { Time } from '@inngest/components/Time';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';

import NotFound from '@/components/Error/NotFound';
import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

import { compactSandboxID, formatTimeout, sandboxStatus } from './utils';
import { SandboxMetrics } from './SandboxMetrics';

const SandboxDetailDocument = graphql(`
  query SandboxDetail($envSlug: String!, $sandboxID: UUID!) {
    envBySlug(slug: $envSlug) {
      sandbox(id: $sandboxID) {
        id
        name
        status
        imageID
        vcpu
        memoryMB
        networkRateLimitMBPS
        timeoutSeconds
        vpcID
        privateIPv4
        macAddress
        command
        environmentVariableNames
        createdAt
        startedAt
        endedAt
        failedAt
        launchUnknownAt
      }
    }
  }
`);

export function SandboxDetail({ sandboxID }: { sandboxID: string }) {
  const environment = useEnvironment();
  const { handleCopyClick, isCopying } = useCopyToClipboard();
  const { data, error, isLoading, refetch } = useGraphQLQuery({
    query: SandboxDetailDocument,
    variables: { envSlug: environment.slug, sandboxID },
  });
  const sandbox = data?.envBySlug?.sandbox;

  if (!isLoading && !error && !sandbox) return <NotFound />;

  return (
    <div className="flex h-full min-h-0 flex-col">
      <Header
        loading={isLoading}
        breadcrumb={[
          {
            text: 'Sandboxes',
            href: pathCreator.sandboxes({ envSlug: environment.slug }),
          },
          { text: sandbox?.name ?? compactSandboxID(sandboxID) },
        ]}
        infoIcon={
          <CopyButton
            code={sandboxID}
            iconOnly
            size="small"
            isCopying={isCopying}
            handleCopyClick={handleCopyClick}
          />
        }
      />

      {error ? (
        <div className="p-6">
          <ErrorCard error={error} reset={() => refetch()} />
        </div>
      ) : sandbox ? (
        <main className="bg-canvasBase min-h-0 flex-1 overflow-y-auto">
          <div className="mx-auto w-full max-w-[1200px] px-6 py-5">
            <div className="mb-5 flex items-center gap-3">
              <div className="min-w-0">
                <h1 className="text-basis truncate text-xl font-medium">
                  {sandbox.name}
                </h1>
                <span className="text-light font-mono text-xs">
                  {compactSandboxID(sandbox.id)}
                </span>
              </div>
              <div className="ml-2">
                <StatusCell
                  status={sandboxStatus(sandbox.status).colorStatus}
                  label={sandboxStatus(sandbox.status).label}
                />
              </div>
            </div>

            <SandboxMetrics
              sandboxID={sandbox.id}
              vcpu={sandbox.vcpu}
              memoryMB={sandbox.memoryMB}
            />

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
              <section>
                <h2 className="text-basis mb-2 text-sm font-medium">
                  Configuration
                </h2>
                <div className="border-subtle overflow-hidden rounded-md border">
                  <DetailRow label="Image" value={sandbox.imageID} mono />
                  <DetailRow
                    label="Command"
                    value={
                      sandbox.command.length
                        ? sandbox.command.join(' ')
                        : 'Default'
                    }
                    mono
                  />
                  <DetailRow
                    label="Environment variables"
                    value={
                      sandbox.environmentVariableNames.length
                        ? sandbox.environmentVariableNames.join(', ')
                        : 'None'
                    }
                    mono
                  />
                  <DetailRow
                    label="Timeout"
                    value={formatTimeout(sandbox.timeoutSeconds)}
                  />
                </div>
              </section>

              <aside>
                <h2 className="text-basis mb-2 text-sm font-medium">Details</h2>
                <div className="border-subtle overflow-hidden rounded-md border">
                  <DetailRow
                    label="Created"
                    value={<Time value={sandbox.createdAt} />}
                  />
                  {sandbox.startedAt && (
                    <DetailRow
                      label="Started"
                      value={<Time value={sandbox.startedAt} />}
                    />
                  )}
                  {sandbox.status === 'STOPPED' && sandbox.endedAt && (
                    <DetailRow
                      label="Stopped"
                      value={<Time value={sandbox.endedAt} />}
                    />
                  )}
                  {sandbox.failedAt && (
                    <DetailRow
                      label="Failed"
                      value={<Time value={sandbox.failedAt} />}
                    />
                  )}
                  {sandbox.launchUnknownAt && (
                    <DetailRow
                      label="Launch result unknown"
                      value={<Time value={sandbox.launchUnknownAt} />}
                    />
                  )}
                  <DetailRow
                    label="Private IP"
                    value={sandbox.privateIPv4}
                    mono
                  />
                  <DetailRow label="VPC" value={sandbox.vpcID} mono />
                  <DetailRow
                    label="MAC address"
                    value={sandbox.macAddress}
                    mono
                  />
                  {sandbox.networkRateLimitMBPS > 0 && (
                    <DetailRow
                      label="Network limit"
                      value={`${sandbox.networkRateLimitMBPS} Mbps`}
                    />
                  )}
                </div>
              </aside>
            </div>
          </div>
        </main>
      ) : null}
    </div>
  );
}

function DetailRow({
  label,
  value,
  mono = false,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <div className="border-subtle grid grid-cols-[140px_minmax(0,1fr)] gap-3 border-b px-4 py-3 last:border-b-0">
      <span className="text-muted text-sm">{label}</span>
      <span
        className={`text-basis min-w-0 break-words text-sm ${mono ? 'font-mono' : ''}`}
      >
        {value}
      </span>
    </div>
  );
}
