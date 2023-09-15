import useCopyToClipboard from '@/hooks/useCopyToClipboard';
import { IconCheck, IconCopy } from '@/icons';
import Button from './Button';

type ButtonCopyProps = {
  code: string;
  iconOnly?: boolean;
  isCopying: boolean;
  handleCopyClick: (code: string) => void;
};

export default function CopyButton({ code, iconOnly, isCopying, handleCopyClick }: ButtonCopyProps) {
  const icon = isCopying ? (
    <IconCheck className="text-teal-500 icon-2xl" />
  ) : (
    <IconCopy className="text-slate-500 icon-2xl" />
  );
  const label = isCopying ? 'Copied!' : 'Copy';

  return (
    <Button
      kind={isCopying ? 'success' : 'default'}
      btnAction={() => handleCopyClick(code)}
      label={iconOnly ? undefined : label}
      appearance={iconOnly ? 'text' : 'solid'}
      icon={iconOnly && icon}
      size={iconOnly ? 'large' : 'small'}
    />
  );
}
