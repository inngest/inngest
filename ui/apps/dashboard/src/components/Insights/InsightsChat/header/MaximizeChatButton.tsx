import { Button } from '@inngest/components/Button';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { RiChat3Line } from '@remixicon/react';

function MaximizeButtonLabel() {
  return (
    <div className="flex items-center">
      <div className="flex rounded-[4px] pr-1">
        <RiChat3Line className="text-muted h-4 w-4" />
      </div>
      <span>Insights AI</span>
    </div>
  );
}

export const MaximizeChatButton = ({
  className,
  onClick,
}: {
  className?: string;
  onClick: () => void;
}) => {
  return (
    <OptionalTooltip tooltip="Show chat" side="bottom">
      <Button
        kind="secondary"
        label={<MaximizeButtonLabel />}
        appearance="ghost"
        className={className}
        onClick={onClick}
      />
    </OptionalTooltip>
  );
};
