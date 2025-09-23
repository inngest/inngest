import { Button } from '@inngest/components/Button';
import { RiArrowUpLine } from '@remixicon/react';

type SendButtonProps = {
  onClick: (e: React.FormEvent) => void;
  className?: string;
  disabled?: boolean;
};

const SendButton = ({ onClick, className, disabled }: SendButtonProps) => {
  return (
    <Button
      kind="secondary"
      appearance="outlined"
      size="small"
      icon={<RiArrowUpLine />}
      onClick={onClick}
      className={className}
      disabled={disabled}
    />
  );
};

export default SendButton;
