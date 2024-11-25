import { NewButton } from '@inngest/components/Button';
import { RiArrowLeftLine } from '@remixicon/react';

import { OptionalTooltip } from '../Tooltip/OptionalTooltip';

export const Back = ({ className }: { className?: string }) => {
  return (
    <OptionalTooltip tooltip="Back to environment" side="right">
      <NewButton
        kind="secondary"
        appearance="outlined"
        size="small"
        icon={<RiArrowLeftLine />}
        className={className}
        href="/"
      />
    </OptionalTooltip>
  );
};
