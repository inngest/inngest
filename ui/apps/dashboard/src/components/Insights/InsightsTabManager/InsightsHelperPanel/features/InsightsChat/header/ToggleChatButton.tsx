import { Button } from "@inngest/components/Button";
import { OptionalTooltip } from "@inngest/components/Tooltip/OptionalTooltip";
import { RiContractRightLine } from "@remixicon/react";

export const ToggleChatButton = ({
  className,
  onClick,
}: {
  className?: string;
  onClick: () => void;
}) => {
  return (
    <OptionalTooltip tooltip="Minimize chat" side="bottom">
      <Button
        kind="secondary"
        appearance="ghost"
        size="small"
        icon={<RiContractRightLine className="text-muted" />}
        className={className}
        onClick={onClick}
      />
    </OptionalTooltip>
  );
};
