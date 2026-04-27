import { Card } from '@inngest/components/Card';
import { Pill } from '@inngest/components/Pill';
import {
  isActive,
  type ExperimentDetail,
} from '@inngest/components/Experiments';
import { cn } from '@inngest/components/utils/classNames';
import { RiFlaskLine, RiScalesLine, RiTrophyLine } from '@remixicon/react';

function formatDuration(from: Date): string {
  const ms = Date.now() - from.getTime();
  const minutes = Math.floor(ms / 60_000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) {
    const remHours = hours % 24;
    return remHours > 0 ? `${days}d ${remHours}h` : `${days}d`;
  }
  if (hours > 0) {
    const remMinutes = minutes % 60;
    return remMinutes > 0 ? `${hours}h ${remMinutes}m` : `${hours}h`;
  }
  return `${minutes}m`;
}

type Props = {
  detail: ExperimentDetail;
  topVariantName: string | null;
};

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-muted mb-2 text-xs font-medium uppercase tracking-wide">
      {children}
    </p>
  );
}

export function InfoSidebar({ detail, topVariantName }: Props) {
  const active = isActive(detail.lastSeen);

  return (
    <div className="flex min-w-[300px] flex-col gap-5 p-4">
      <section>
        <SectionLabel>Overview</SectionLabel>
        <Card>
          <Card.Content className="flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <RiFlaskLine className="text-muted h-4 w-4 shrink-0" />
              <span className="text-basis truncate text-sm font-medium">
                {detail.name}
              </span>
            </div>
            <p className="text-muted text-xs">
              Started at {detail.firstSeen.toLocaleString()}
            </p>
            <p className="text-muted text-xs">
              Running {formatDuration(detail.firstSeen)}
            </p>
            <p className="text-muted text-xs">
              {detail.variants
                .reduce((sum, v) => sum + v.runCount, 0)
                .toLocaleString()}{' '}
              total runs
            </p>
            {active && (
              <Pill kind="primary" appearance="outlined">
                Active
              </Pill>
            )}
          </Card.Content>
        </Card>
      </section>

      <section>
        <SectionLabel>Type</SectionLabel>
        <Card>
          <Card.Content className="flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <RiScalesLine className="text-muted h-4 w-4 shrink-0" />
              <span className="text-basis text-sm">
                {detail.selectionStrategy}
              </span>
            </div>
            <Pill kind="default" appearance="solid">
              {detail.variants.length} variant
              {detail.variants.length !== 1 ? 's' : ''}
            </Pill>
          </Card.Content>
        </Card>
      </section>

      <section>
        <SectionLabel>Variants</SectionLabel>
        <Card>
          <Card.Content className="flex flex-col gap-1">
            {detail.variants.map((v) => {
              const weight = detail.variantWeights.find(
                (w) => w.variantName === v.variantName,
              );
              const isTop = v.variantName === topVariantName;

              return (
                <div
                  key={v.variantName}
                  className={cn(
                    'flex items-center justify-between rounded px-2 py-1.5 text-sm',
                    isTop ? 'bg-primary-3xSubtle' : '',
                  )}
                >
                  <div className="flex items-center gap-1.5">
                    <span
                      className={cn(
                        'truncate',
                        isTop ? 'text-primary-intense' : 'text-basis',
                      )}
                    >
                      {v.variantName}
                    </span>
                    {isTop && (
                      <RiTrophyLine className="text-primary-intense h-3.5 w-3.5 shrink-0" />
                    )}
                  </div>
                  {weight != null && (
                    <span className="text-muted tabular-nums text-xs">
                      {weight.weight}
                    </span>
                  )}
                </div>
              );
            })}
          </Card.Content>
        </Card>
      </section>
    </div>
  );
}
