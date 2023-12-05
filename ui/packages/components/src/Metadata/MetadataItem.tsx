import { Badge } from '@inngest/components/Badge';
import { Skeleton } from '@inngest/components/Skeleton';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { IconInfo } from '@inngest/components/icons/Info';
import { classNames } from '@inngest/components/utils/classNames';

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
    <div className={classNames('flex flex-col-reverse p-1.5', className)}>
      <dt className="flex items-center gap-1">
        <span className="text-sm capitalize text-slate-400 dark:text-slate-500">{label}</span>
        {tooltip && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                {/* Temporarily breaks accessibility https://github.com/radix-ui/primitives/discussions/560 */}
                <span>
                  <IconInfo className="h-4 w-4 text-slate-400" />
                </span>
              </TooltipTrigger>
              <TooltipContent className="whitespace-pre-line">{tooltip}</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}
      </dt>
      <dd className="flex justify-between gap-2">
        {loading ? (
          <Skeleton className="h-5 w-full" />
        ) : (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger
                className={classNames(
                  type === 'code' && 'font-mono',
                  'truncate text-sm text-slate-800 dark:text-white'
                )}
              >
                <span
                  className={classNames(
                    type === 'code' && 'font-mono',
                    'truncate text-sm text-slate-800 dark:text-white'
                  )}
                >
                  {value}
                </span>
              </TooltipTrigger>
              <TooltipContent className={classNames(type === 'code' && 'font-mono', 'text-xs')}>
                {title || `${value}`}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}
        {badge && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Badge className="!px-1.5 !py-1">{badge.label}</Badge>
                </span>
              </TooltipTrigger>
              {badge.description && <TooltipContent>{badge.description}</TooltipContent>}
            </Tooltip>
          </TooltipProvider>
        )}
      </dd>
    </div>
  );
}
