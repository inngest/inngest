import { CopyButton } from '@inngest/components/CopyButton';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { cn } from '@inngest/components/utils/classNames';

type CodeLineProps = {
  code: string;
  className?: string;
};

export function CodeLine({ code, className }: CodeLineProps) {
  const { handleCopyClick, isCopying } = useCopyToClipboard();

  return (
    <div
      className={cn(
        className,
        'bg-codeEditor flex items-center justify-between rounded-md text-sm'
      )}
    >
      <code
        className="text-codeDelimiterBracketJson flex-1 cursor-pointer px-4 py-2"
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
