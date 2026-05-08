import { Card } from '@inngest/components/Card';
import { Pill } from '@inngest/components/Pill';
import {
  formatVariantWeight,
  isActive,
  type ExperimentDetail,
  type ExperimentVariantMetrics,
} from '@inngest/components/Experiments';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiArrowLeftLine,
  RiFlaskLine,
  RiScalesLine,
  RiTrophyLine,
} from '@remixicon/react';

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

function IconTile({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-canvasSubtle border-subtle flex h-9 w-9 shrink-0 items-center justify-center rounded border">
      {children}
    </div>
  );
}

export function InfoSidebar({ detail, topVariantName }: Props) {
  const active = isActive(detail.lastSeen);

  return (
    <div className="flex min-w-[300px] flex-col gap-5 p-4">
      <section>
        <SectionLabel>Overview</SectionLabel>
        <Card>
          <Card.Content className="flex flex-col gap-2 p-2">
            <div className="flex items-center gap-2">
              <IconTile>
                <RiFlaskLine className="text-muted h-[18px] w-[18px]" />
              </IconTile>
              <div className="flex min-w-0 flex-1 flex-col">
                <span className="text-basis truncate text-sm font-medium">
                  {detail.name}
                </span>
                <span className="text-muted truncate text-xs">
                  Running {formatDuration(detail.firstSeen)}
                </span>
              </div>
              {active && (
                <Pill kind="primary" appearance="outlined">
                  Active
                </Pill>
              )}
            </div>
            <p className="text-muted text-xs">
              Started at {detail.firstSeen.toLocaleString()}
            </p>
            <p className="text-muted text-xs">
              {detail.variants
                .reduce((sum, v) => sum + v.runCount, 0)
                .toLocaleString()}{' '}
              total runs
            </p>
          </Card.Content>
        </Card>
      </section>

      <section>
        <SectionLabel>Type</SectionLabel>
        <Card>
          <Card.Content className="flex flex-col gap-2 p-2">
            <div className="flex items-center gap-2">
              <IconTile>
                <RiScalesLine className="text-muted h-[18px] w-[18px]" />
              </IconTile>
              <span className="text-basis min-w-0 flex-1 truncate text-sm">
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

      <VariantsSection detail={detail} topVariantName={topVariantName} />
    </div>
  );
}

function VariantsSection({
  detail,
  topVariantName,
}: {
  detail: ExperimentDetail;
  topVariantName: string | null;
}) {
  const isFixed = detail.selectionStrategy === 'fixed';
  const hasWeights = detail.variantWeights.length > 0;

  // For fixed experiments, the "selected" variant is the one that actually
  // received traffic — fall back to the first variant if nothing has run yet.
  const selectedFixedVariant = isFixed
    ? detail.variants.reduce<ExperimentVariantMetrics | null>(
        (top, v) => (v.runCount > (top?.runCount ?? -1) ? v : top),
        null,
      )
    : null;

  return (
    <section>
      <div className="mb-2 flex items-center justify-between">
        <SectionLabel>Variants ({detail.variants.length})</SectionLabel>
        {!isFixed && hasWeights && (
          <span className="text-light text-[11px] font-normal">Weight</span>
        )}
      </div>

      <div className="border-subtle overflow-hidden rounded-md border">
        {detail.variants.map((v, i) => {
          const weight = detail.variantWeights.find(
            (w) => w.variantName === v.variantName,
          );
          const isTop = v.variantName === topVariantName;
          const isSelected =
            selectedFixedVariant?.variantName === v.variantName;
          const isLast = i === detail.variants.length - 1;

          if (isFixed) {
            return (
              <div
                key={v.variantName}
                className={cn(
                  'flex h-8 items-center gap-2 px-2 text-sm',
                  !isLast && 'border-subtle border-b',
                  !isSelected && 'bg-canvasSubtle text-disabled',
                )}
              >
                <span
                  className={cn(
                    'min-w-0 flex-1 truncate',
                    isSelected ? 'text-muted' : 'text-disabled',
                  )}
                  title={v.variantName}
                >
                  {v.variantName}
                </span>
                {isSelected && (
                  <RiArrowLeftLine className="text-muted h-4 w-4 shrink-0" />
                )}
              </div>
            );
          }

          return (
            <div
              key={v.variantName}
              className={cn(
                'flex h-8 items-center gap-2 px-2 text-sm',
                !isLast && 'border-subtle border-b',
              )}
            >
              <span
                className="text-muted min-w-0 flex-1 truncate"
                title={v.variantName}
              >
                {v.variantName}
              </span>
              {isTop && (
                <Pill
                  kind="primary"
                  appearance="solidBright"
                  icon={<RiTrophyLine className="h-3 w-3" />}
                  iconSide="iconOnly"
                >
                  {null}
                </Pill>
              )}
              {weight != null && (
                <span className="text-basis font-mono text-sm tabular-nums">
                  {formatVariantWeight(weight.weight)}
                </span>
              )}
            </div>
          );
        })}
      </div>
    </section>
  );
}
