import { IconExclamationTriangle } from '@/icons';
import classNames from '@/utils/classnames';

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
          'min-w-[420px] bg-slate-800 rounded-md text-slate-300 py-2 px-4',
          isInvalid && value.length > 0 && 'pr-8 border-rose-400 border-2',
          className,
        )}
        value={value}
        onChange={onChange}
        {...props}
      />
      {isInvalid && value.length > 0 && (
        <IconExclamationTriangle className="absolute top-2/4 right-2 -translate-y-2/4 text-rose-400" />
      )}
    </div>
  );
}
