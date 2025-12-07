import { Button } from "@inngest/components/Button/NewButton";
import { RiArrowUpLine } from "@remixicon/react";

type SendButtonProps = {
  onClick: (e: React.FormEvent) => void;
  disabled?: boolean;
};

const SendButton = ({ onClick, disabled }: SendButtonProps) => {
  return (
    <Button
      kind="primary"
      size="small"
      icon={<RiArrowUpLine className="text-white" />}
      onClick={onClick}
      disabled={disabled}
    />
  );
};

export default SendButton;
