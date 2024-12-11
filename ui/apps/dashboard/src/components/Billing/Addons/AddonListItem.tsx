import { NewButton } from '@inngest/components/Button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiInformationLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';

export default function AddOn({
  title,
  description,
  value,
  canIncreaseLimitInCurrentPlan,
  tooltipContent,
}: {
  title: string;
  description?: string;
  value?: number | string;
  canIncreaseLimitInCurrentPlan: boolean;
  tooltipContent?: string | React.ReactNode;
}) {
  return (
    <div className="mb-6 flex items-start justify-between text-right">
      <div>
        <p className="text-subtle mb-1 flex items-center gap-1 text-xs font-medium">
          {title}
          {tooltipContent && (
            <Tooltip>
              <TooltipTrigger>
                <RiInformationLine className="text-light h-4 w-4" />
              </TooltipTrigger>
              <TooltipContent className="whitespace-pre-line">{tooltipContent}</TooltipContent>
            </Tooltip>
          )}
        </p>
        {description && <p className="text-subtle mb-2 text-xs italic">{description}</p>}
      </div>
      <div>
        {value && <p className="text-basis pr-3 text-sm font-medium">{value}</p>}
        <NewButton
          appearance="ghost"
          label={canIncreaseLimitInCurrentPlan ? 'Contact for more' : 'Upgrade for more'}
          href={
            canIncreaseLimitInCurrentPlan
              ? pathCreator.support({
                  ref: `app-billing-page-overview-addon-${title.toLowerCase().replace(/ /g, '-')}`,
                })
              : pathCreator.billing({
                  tab: 'plans',
                  ref: `app-billing-page-overview-addon-${title.toLowerCase().replace(/ /g, '-')}`,
                })
          }
        />
      </div>
    </div>
  );
}
