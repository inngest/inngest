import { NewButton } from '@inngest/components/Button';
import { RiArrowLeftLine } from '@remixicon/react';

import { OptionalTooltip } from '../Navigation/OptionalTooltip';

export const Back = ({ className }: { className?: string }) => {
  return (
    <OptionalTooltip tooltip="Back to environment">
      <NewButton
        kind="secondary"
        appearance="outlined"
        size="small"
        icon={<RiArrowLeftLine />}
        className={className}
        href="/"
        scroll={false}
      />
    </OptionalTooltip>
  );
};
