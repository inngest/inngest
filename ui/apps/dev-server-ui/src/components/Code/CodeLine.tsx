import { CopyButton } from '@inngest/components/CopyButton';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { classNames } from '@inngest/components/utils/classNames';

type CodeLineProps = {
  code: string;
  className?: string;
};

export default function CodeLine({ code, className }: CodeLineProps) {
  const { handleCopyClick, isCopying } = useCopyToClipboard();

  return (
    <div
      className={classNames(
        className,
        'bg-slate-910 flex cursor-pointer items-center justify-between rounded-md'
      )}
      onClick={() => handleCopyClick(code)}
    >
      <code className="text-slate-300">{code}</code>
      <CopyButton
        code={code}
        iconOnly={true}
        isCopying={isCopying}
        handleCopyClick={handleCopyClick}
      />
    </div>
  );
}
