import { Button } from '../Button';
import { Card } from '../Card';
import { CodeBlock } from '../CodeBlock';
import { Time } from '../Time';
import { toMaybeDate } from '../TimelineV2/utils';
import { cn } from '../utils/classNames';
import { formatMilliseconds } from '../utils/date';

type Props = {
  className?: string;
  app: {
    name: string;
  };
  fn: {
    name: string;
  };
  run: {
    id: string;
    output: string | null;
    trace: {
      childrenSpans?: unknown[];
      endedAt: string | null;
      queuedAt: string;
      startedAt: string | null;
    };
  };
};

export function RunInfo({ className, app, fn, run }: Props) {
  const queuedAt = new Date(run.trace.queuedAt);
  const startedAt = toMaybeDate(run.trace.startedAt);
  const endedAt = toMaybeDate(run.trace.endedAt);

  const delayText = formatMilliseconds((startedAt ?? new Date()).getTime() - queuedAt.getTime());

  let durationText = '-';
  if (startedAt) {
    durationText = formatMilliseconds((endedAt ?? new Date()).getTime() - startedAt.getTime());
  }

  return (
    <div className={cn('flex flex-col gap-4', className)}>
      <Card>
        <Card.Header className="flex-row items-center gap-2">
          <div className="grow">Run details</div>

          <Button label="Cancel" size="small" />
          <Button label="Rerun" size="small" />
          <Button label="Rerun in Dev Server" size="small" />
        </Card.Header>

        <Card.Content>
          <div>
            <dl className="flex flex-wrap gap-2">
              <Labeled label="App">{app.name}</Labeled>

              <Labeled label="Function">{fn.name}</Labeled>

              <Labeled label="Run ID">
                <span className="font-mono">{run.id}</span>
              </Labeled>

              <Labeled label="Trigger">foo</Labeled>

              <Labeled label="Event received at"></Labeled>

              <Labeled label="Queued at">
                <Time value={queuedAt} />
              </Labeled>

              <Labeled label="Started at">{startedAt ? <Time value={startedAt} /> : '-'}</Labeled>

              <Labeled label="Ended at">{endedAt ? <Time value={endedAt} /> : '-'}</Labeled>

              <Labeled label="Delay">{delayText}</Labeled>

              <Labeled label="Duration">{durationText}</Labeled>

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
      <dt className="text-slate-500">{label}</dt>
      <dd className="truncate">{children}</dd>
    </div>
  );
}
