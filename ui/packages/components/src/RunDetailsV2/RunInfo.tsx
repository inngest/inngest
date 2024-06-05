import type { Route } from 'next';

import { CancelRunButton } from '../CancelRunButton';
import { Card } from '../Card';
import {
  ElementWrapper,
  IDElement,
  LinkElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';
import { Link } from '../Link';
import { RerunButton } from '../RerunButtonV2';
import { cn } from '../utils/classNames';
import { formatMilliseconds, toMaybeDate } from '../utils/date';

type Props = {
  standalone: boolean;
  cancelRun: () => Promise<unknown>;
  className?: string;
  app: {
    externalID: string;
    name: string;
  };
  fn: {
    id: string;
    name: string;
  };
  pathCreator: {
    app: (params: { externalAppID: string }) => Route;
    runPopout: (params: { runID: string }) => Route;
  };
  rerun: (args: { fnID: string }) => Promise<unknown>;
  run: {
    id: string;
    trace: {
      childrenSpans?: unknown[];
      endedAt: string | null;
      queuedAt: string;
      startedAt: string | null;
      status: string;
    };
  };
};

export function RunInfo({
  app,
  cancelRun,
  className,
  fn,
  pathCreator,
  rerun,
  run,
  standalone,
}: Props) {
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
        <Card.Header className="h-11 flex-row items-center gap-2">
          <div className="flex grow items-center gap-2">
            Run details {!standalone && <Link href={pathCreator.runPopout({ runID: run.id })} />}
          </div>

          <CancelRunButton disabled={Boolean(endedAt)} onClick={cancelRun} />
          <RerunButton onClick={() => rerun({ fnID: fn.id })} />
        </Card.Header>

        <Card.Content>
          <div>
            <dl className="flex flex-wrap gap-4">
              <ElementWrapper label="Run ID">
                <IDElement>{run.id}</IDElement>
              </ElementWrapper>

              <ElementWrapper label="App">
                <LinkElement
                  internalNavigation
                  href={pathCreator.app({ externalAppID: app.externalID })}
                  showIcon={false}
                >
                  {app.name}
                </LinkElement>
              </ElementWrapper>

              <ElementWrapper label="Duration">
                <TextElement>{durationText}</TextElement>
              </ElementWrapper>

              <ElementWrapper label="Queued at">
                <TimeElement date={queuedAt} />
              </ElementWrapper>

              <ElementWrapper label="Started at">
                {startedAt ? <TimeElement date={startedAt} /> : <TextElement>-</TextElement>}
              </ElementWrapper>

              <ElementWrapper label="Ended at">
                {endedAt ? <TimeElement date={endedAt} /> : <TextElement>-</TextElement>}
              </ElementWrapper>

              <ElementWrapper label="Step count">
                <TextElement>{run.trace.childrenSpans?.length ?? 0}</TextElement>
              </ElementWrapper>
            </dl>
          </div>
        </Card.Content>
      </Card>
    </div>
  );
}
