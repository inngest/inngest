import { Card } from '@inngest/components/Card';
import { Link } from '@inngest/components/Link/Link';
import { Pill } from '@inngest/components/Pill';
import {
  formatVariantWeight,
  isActive,
  type ExperimentDetail,
  type ExperimentVariantMetrics,
} from '@inngest/components/Experiments';
import { truncateCenter } from '@/lib/experiments/chart';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiArrowLeftLine,
  RiExternalLinkLine,
  RiFlaskLine,
  RiScalesLine,
  RiTrophyLine,
} from '@remixicon/react';

type Props = {
  detail: ExperimentDetail;
  topVariantName: string | null;
  variantOrder?: string[];
  functionName: string;
  functionHref: string;
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

export function InfoSidebar({
  detail,
  topVariantName, variantOrder, functionName, functionHref, }: Props) {
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
              </div>
              {active && (
                <Pill kind="primary" appearance="outlined">
                  Active
                </Pill>
              )}
            </div>
          </Card.Content>
        </Card>
      </section>

      <section>
        <SectionLabel>Function</SectionLabel>
        <Card>
          <Card.Content className="flex items-center gap-2 p-2">
            <IconTile>
              <FunctionsIcon className="text-muted h-[18px] w-[18px]" />
            </IconTile>
            <span
              className="text-basis min-w-0 flex-1 truncate text-sm"
              title={functionName}
            >
              {functionName}
            </span>
            <Link
              href={functionHref}
              iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
            >
              View
            </Link>
          </Card.Content>
        </Card>
      </section>

      <section>
        <SectionLabel>Type</SectionLabel>
        <Card>
          <Card.Content className="flex items-center gap-2 p-2">
            <IconTile>
              <RiScalesLine className="text-muted h-[18px] w-[18px]" />
            </IconTile>
            <span className="text-basis min-w-0 flex-1 truncate text-sm">
              {detail.selectionStrategy}
            </span>
          </Card.Content>
        </Card>
      </section>

      <VariantsSection detail={detail} topVariantName={topVariantName} variantOrder={variantOrder} />
    </div>
  );
}

function VariantsSection({
  detail,
  topVariantName,
  variantOrder,
}: {
  detail: ExperimentDetail;
  topVariantName: string | null;
  variantOrder?: string[];
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

  const sortedVariants = variantOrder
    ? [...detail.variants].sort((a, b) => {
        const ai = variantOrder.indexOf(a.variantName);
        const bi = variantOrder.indexOf(b.variantName);
        return (ai === -1 ? Infinity : ai) - (bi === -1 ? Infinity : bi);
      })
    : detail.variants;

  return (
    <section>
      <div className="mb-2 flex items-center justify-between">
        <SectionLabel>Variants ({detail.variants.length})</SectionLabel>
        {!isFixed && hasWeights && (
          <span className="text-light text-[11px] font-normal">Weight</span>
        )}
      </div>

      <div className="border-subtle overflow-hidden rounded-md border">
        {sortedVariants.map((v, i) => {
          const weight = detail.variantWeights.find(
            (w) => w.variantName === v.variantName,
          );
          const isTop = v.variantName === topVariantName;
          const isSelected =
            selectedFixedVariant?.variantName === v.variantName;
          const isLast = i === sortedVariants.length - 1;

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
                  {truncateCenter(v.variantName)}
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
              <div className="flex min-w-0 flex-1 items-center gap-1.5">
                <span
                  className="text-muted truncate"
                  title={v.variantName}
                >
                  {truncateCenter(v.variantName)}
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
              </div>
              <span className="text-basis font-mono text-sm tabular-nums">
               {weight != null ? formatVariantWeight(weight.weight): "-"}
              </span>
            </div>
          );
        })}
      </div>
    </section>
  );
}
