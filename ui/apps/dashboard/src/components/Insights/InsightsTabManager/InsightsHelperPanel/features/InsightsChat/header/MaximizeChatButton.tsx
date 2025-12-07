import { Button } from "@inngest/components/Button/NewButton";
import { OptionalTooltip } from "@inngest/components/Tooltip/OptionalTooltip";
import { RiChat3Line } from "@remixicon/react";

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
        icon={<RiChat3Line className="text-muted" />}
        label="Insights AI"
        iconSide="left"
        appearance="ghost"
        className={className}
        onClick={onClick}
      />
    </OptionalTooltip>
  );
};
