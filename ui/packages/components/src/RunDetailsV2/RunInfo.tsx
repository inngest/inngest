import type { UrlObject } from 'url';
import type { Route } from 'next';

import { CancelRunButton } from '../CancelRunButton';
import { Card } from '../Card';
import { CodeBlock } from '../CodeBlock';
import { Link } from '../Link';
import { RerunButton } from '../RerunButtonV2';
import { Time } from '../Time';
import { cn } from '../utils/classNames';
import { formatMilliseconds, toMaybeDate } from '../utils/date';

type Props = {
  standalone: boolean;
  cancelRun: () => Promise<unknown>;
  className?: string;
  app: {
    name: string;
    url: Route | UrlObject;
  };
  fn: {
    id: string;
    name: string;
  };
  rerun: (args: { fnID: string }) => Promise<unknown>;
  run: {
    id: string;
    output: string | null;
    url: Route | UrlObject;
    trace: {
      childrenSpans?: unknown[];
      endedAt: string | null;
      queuedAt: string;
      startedAt: string | null;
      status: string;
    };
  };
};

export function RunInfo({ app, cancelRun, className, fn, rerun, run, standalone }: Props) {
  const queuedAt = new Date(run.trace.queuedAt);
  const startedAt = toMaybeDate(run.trace.startedAt);
  const endedAt = toMaybeDate(run.trace.endedAt);

  let durationText = '-';
  if (startedAt) {
    durationText = formatMilliseconds((endedAt ?? new Date()).getTime() - startedAt.getTime());
  }

  return (
    <div className={cn('flex flex-col gap-5', className)}>
      <Card>
        <Card.Header className="flex-row items-center gap-2">
          <div className="flex grow items-center gap-2">
            Run details {!standalone && <Link href={run.url} />}
          </div>

          <CancelRunButton disabled={Boolean(endedAt)} onClick={cancelRun} />
          <RerunButton onClick={() => rerun({ fnID: fn.id })} />
        </Card.Header>

        <Card.Content>
          <div>
            <dl className="flex flex-wrap gap-4">
              <Labeled label="Run ID">
                <span className="font-mono">{run.id}</span>
              </Labeled>

              <Labeled label="App">
                <Link internalNavigation href={app.url} showIcon={false}>
                  {app.name}
                </Link>
              </Labeled>

              <Labeled label="Duration">{durationText}</Labeled>

              <Labeled label="Queued at">
                <Time value={queuedAt} />
              </Labeled>

              <Labeled label="Started at">{startedAt ? <Time value={startedAt} /> : '-'}</Labeled>

              <Labeled label="Ended at">{endedAt ? <Time value={endedAt} /> : '-'}</Labeled>

              <Labeled label="Step count">{run.trace.childrenSpans?.length ?? 0}</Labeled>
            </dl>
          </div>
        </Card.Content>
      </Card>

      {run.output && (
        <CodeBlock
          tabs={[
            {
              label: 'Run output',
              content: run.output,
            },
          ]}
        />
      )}
    </div>
  );
}

function Labeled({ label, children }: React.PropsWithChildren<{ label: string }>) {
  return (
    <div className="w-64 text-sm">
      <dt className="pb-2 text-slate-500">{label}</dt>
      <dd className="truncate">{children}</dd>
    </div>
  );
}
