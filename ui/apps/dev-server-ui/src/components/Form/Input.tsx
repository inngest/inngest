import { classNames } from '@inngest/components/utils/classNames';

import { IconExclamationTriangle } from '@/icons';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  value: string;
  className?: string;
  isInvalid?: boolean;
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
}

export default function Input({ value, className, onChange, isInvalid, ...props }: InputProps) {
  return (
    <div className="relative">
      <input
        id="addAppUrlModal"
        className={classNames(
          'min-w-[420px] rounded-md bg-slate-800 px-4 py-2 text-slate-300',
          isInvalid && value.length > 0 && 'border-2 border-rose-400 pr-8',
          className
        )}
        value={value}
        onChange={onChange}
        {...props}
      />
      {isInvalid && value.length > 0 && (
        <IconExclamationTriangle className="absolute right-2 top-2/4 -translate-y-2/4 text-rose-400" />
      )}
    </div>
  );
}
