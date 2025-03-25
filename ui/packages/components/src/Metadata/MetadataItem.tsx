import { Pill } from '@inngest/components/Pill';
import { Skeleton } from '@inngest/components/Skeleton';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiInformationLine } from '@remixicon/react';

export type MetadataItemProps = {
  className?: string;
  label: string;
  value: string | JSX.Element;
  title?: string;
  tooltip?: string;
  badge?: {
    label: string;
    description?: string;
  };
  type?: 'code' | 'text';
  size?: 'small' | 'large';
  loading?: boolean;
};

export function MetadataItem({
  className,
  value,
  title,
  label,
  type,
  tooltip,
  badge,
  loading = false,
}: MetadataItemProps) {
  return (
    <div className={cn('flex flex-col-reverse p-1.5', className)}>
      <dt className="flex items-center gap-1">
        <span className="text-subtle text-sm capitalize">{label}</span>
        {tooltip && (
          <Tooltip>
            <TooltipTrigger asChild>
              {/* Temporarily breaks accessibility https://github.com/radix-ui/primitives/discussions/560 */}
              <span>
                <RiInformationLine className="text-muted h-4 w-4" />
              </span>
            </TooltipTrigger>
            <TooltipContent className="whitespace-pre-line">{tooltip}</TooltipContent>
          </Tooltip>
        )}
      </dt>
      <dd className="flex justify-between gap-2">
        {loading ? (
          <Skeleton className="h-5 w-full" />
        ) : (
          <Tooltip>
            <TooltipTrigger
              className={cn(type === 'code' && 'font-mono', 'text-basis truncate text-sm')}
            >
              <span className={cn(type === 'code' && 'font-mono', 'text-basis truncate text-sm')}>
                {value}
              </span>
            </TooltipTrigger>
            <TooltipContent className={cn(type === 'code' && 'font-mono', 'text-xs')}>
              {title || `${value}`}
            </TooltipContent>
          </Tooltip>
        )}
        {badge && (
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Pill appearance="outlined" className="!px-1.5 !py-1">
                  {badge.label}
                </Pill>
              </span>
            </TooltipTrigger>
            {badge.description && <TooltipContent>{badge.description}</TooltipContent>}
          </Tooltip>
        )}
      </dd>
    </div>
  );
}
