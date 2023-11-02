import { CopyButton } from '@inngest/components/CopyButton';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { classNames } from '@inngest/components/utils/classNames';

type CodeLineProps = {
  code: string;
  className?: string;
};

export function CodeLine({ code, className }: CodeLineProps) {
  const { handleCopyClick, isCopying } = useCopyToClipboard();

  return (
    <div
      className={classNames(
        className,
        'dark:bg-slate-910 flex items-center justify-between rounded-md bg-slate-50 text-sm'
      )}
    >
      <code
        className="flex-1 cursor-pointer px-4 py-2 text-slate-500 dark:text-slate-300"
        onClick={() => handleCopyClick(code)}
      >
        {code}
      </code>
      <CopyButton
        code={code}
        iconOnly={true}
        isCopying={isCopying}
        handleCopyClick={handleCopyClick}
      />
    </div>
  );
}
