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
  type?: 'code' | 'text';
  size?: 'small' | 'large';
};

export function MetadataItem({ className, value, title, label, type, tooltip }: MetadataItemProps) {
  return (
    <dl className={classNames('flex flex-col p-1.5', className)}>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <dd
              className={classNames(
                type === 'code' && 'font-mono',
                'truncate text-sm text-slate-800 dark:text-white'
              )}
            >
              {value}
            </dd>
          </TooltipTrigger>
          <TooltipContent className="font-mono text-xs">{title || `${value}`}</TooltipContent>
        </Tooltip>
      </TooltipProvider>

      <dt className="flex items-center gap-1">
        <span className="text-sm capitalize text-slate-400 dark:text-slate-500">{label}</span>
        {tooltip && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger>
                <IconInfo className="h-4 w-4 text-slate-400" />
              </TooltipTrigger>
              <TooltipContent>{tooltip}</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}
      </dt>
    </dl>
  );
}
