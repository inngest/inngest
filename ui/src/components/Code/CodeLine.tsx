import useCopyToClipboard from '@/hooks/useCopyToClipboard';
import classNames from '@/utils/classnames';
import CopyButton from '@/components/Button/CopyButton';

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
        'flex items-center justify-between bg-slate-950 rounded-md cursor-pointer',
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
