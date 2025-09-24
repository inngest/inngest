import { Button } from '@inngest/components/Button';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { RiEraserLine } from '@remixicon/react';

type ClearThreadButtonProps = {
  onClick: () => void;
  className?: string;
};

export default function ClearThreadButton({ onClick, className }: ClearThreadButtonProps) {
  return (
    <OptionalTooltip tooltip="Clear chat" side="bottom">
      <Button
        kind="secondary"
        appearance="ghost"
        size="small"
        icon={<RiEraserLine className="text-muted" />}
        className={className}
        onClick={onClick}
      />
    </OptionalTooltip>
  );
}
