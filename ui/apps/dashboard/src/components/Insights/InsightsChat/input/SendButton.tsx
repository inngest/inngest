import { Button } from '@inngest/components/Button';
import { RiArrowUpLine } from '@remixicon/react';

type SendButtonProps = {
  onClick: (e: React.FormEvent) => void;
  disabled?: boolean;
};

const SendButton = ({ onClick, disabled }: SendButtonProps) => {
  return (
    <Button
      kind="secondary"
      size="small"
      icon={<RiArrowUpLine className="text-white" />}
      onClick={onClick}
      className={`${disabled ? 'cursor-not-allowed bg-[#9ADAB3] opacity-50' : ''} bg-[#007A48]`}
      disabled={disabled}
    />
  );
};

export default SendButton;
