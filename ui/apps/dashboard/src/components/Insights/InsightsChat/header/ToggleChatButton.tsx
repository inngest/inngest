import { Button } from '@inngest/components/Button';
import { RiArrowLeftLine } from '@remixicon/react';

export const ToggleChatButton = ({
  className,
  onClick,
}: {
  className?: string;
  onClick: () => void;
}) => {
  return (
    <Button
      kind="secondary"
      appearance="outlined"
      size="small"
      icon={<RiArrowLeftLine />}
      className={className}
      onClick={onClick}
    />
  );
};
