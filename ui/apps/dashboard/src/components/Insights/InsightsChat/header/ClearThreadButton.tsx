import { Button } from '@inngest/components/Button';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { RiEraserLine } from '@remixicon/react';

type ClearThreadButtonProps = {
  onClick: () => void;
  className?: string;
  style?: React.CSSProperties;
};

export default function ClearThreadButton({ onClick, className, style }: ClearThreadButtonProps) {
  return (
    <OptionalTooltip tooltip="Clear chat" side="bottom">
      <Button
        kind="secondary"
        appearance="ghost"
        size="small"
        icon={<RiEraserLine />}
        className={className}
        onClick={onClick}
        style={style}
      />
    </OptionalTooltip>
  );
}
