import { Button } from '@inngest/components/Button';
import { IconCheck } from '@inngest/components/icons/Check';
import { IconCopy } from '@inngest/components/icons/Copy';

type ButtonCopyProps = {
  code?: string;
  iconOnly?: boolean;
  isCopying: boolean;
  handleCopyClick: (code: string) => void;
  size?: 'small' | 'regular' | 'large';
};

export function CopyButton({ size, code, iconOnly, isCopying, handleCopyClick }: ButtonCopyProps) {
  const icon = isCopying ? <IconCheck /> : <IconCopy />;
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
