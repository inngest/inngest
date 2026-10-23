import { Button } from '@inngest/components/Button';
import { RiCheckLine, RiFileCopy2Line } from '@remixicon/react';

type ButtonCopyProps = {
  code?: string;
  label?: string;
  disabled?: boolean;
  ariaLabel?: string;
  className?: string;
  iconOnly?: boolean;
  isCopying: boolean;
  handleCopyClick: (code: string) => void;
  size?: 'small' | 'medium' | 'large';
  appearance?: 'solid' | 'outlined' | 'ghost';
};

export function CopyButton({
  size,
  ariaLabel = 'Copy',
  className,
  code,
  label: copyLabel = 'Copy',
  disabled,
  iconOnly,
  isCopying,
  handleCopyClick,
  appearance = 'solid',
}: ButtonCopyProps) {
  const icon = isCopying ? <RiCheckLine /> : <RiFileCopy2Line />;
  const label = isCopying ? 'Copied!' : copyLabel;

  return (
    <Button
      className={className}
      disabled={disabled || !code}
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
