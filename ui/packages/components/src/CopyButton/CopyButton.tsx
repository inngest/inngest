import { Button } from '@inngest/components/Button';
import { RiCheckLine, RiFileCopy2Line } from '@remixicon/react';

type ButtonCopyProps = {
  code?: string;
  iconOnly?: boolean;
  isCopying: boolean;
  handleCopyClick: (code: string) => void;
  size?: 'small' | 'regular' | 'large';
};

export function CopyButton({ size, code, iconOnly, isCopying, handleCopyClick }: ButtonCopyProps) {
  const icon = isCopying ? <RiCheckLine /> : <RiFileCopy2Line />;
  const label = isCopying ? 'Copied!' : 'Copy';

  return (
    <Button
      disabled={!code}
      size={size}
      kind={isCopying ? 'success' : 'default'}
      btnAction={code ? () => handleCopyClick(code) : undefined}
      label={iconOnly ? undefined : label}
      appearance={iconOnly ? 'text' : 'solid'}
      icon={iconOnly && icon}
      title="Click to copy"
      aria-label="Copy"
    />
  );
}
