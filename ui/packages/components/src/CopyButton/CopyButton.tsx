import { Button } from '@inngest/components/Button';
import { RiCheckLine, RiFileCopy2Line } from '@remixicon/react';

type ButtonCopyProps = {
  code?: string;
  ariaLabel?: string;
  iconOnly?: boolean;
  isCopying: boolean;
  handleCopyClick: (code: string) => void;
  size?: 'small' | 'medium' | 'large';
  appearance?: 'solid' | 'outlined' | 'ghost';
};

export function CopyButton({
  size,
  ariaLabel = 'Copy',
  code,
  iconOnly,
  isCopying,
  handleCopyClick,
  appearance = 'solid',
}: ButtonCopyProps) {
  const icon = isCopying ? <RiCheckLine /> : <RiFileCopy2Line />;
  const label = isCopying ? 'Copied!' : 'Copy';

  return (
    <Button
      disabled={!code}
      size={size}
      kind={isCopying ? 'primary' : 'secondary'}
      onClick={code ? () => handleCopyClick(code) : undefined}
      label={iconOnly ? undefined : label}
      appearance={iconOnly ? 'ghost' : isCopying ? 'solid' : appearance}
      icon={iconOnly && icon}
      title="Click to copy"
      aria-label={ariaLabel}
    />
  );
}
