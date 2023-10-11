import { Switch } from '@headlessui/react';

import cn from '@/utils/cn';

type Props = {
  checked?: boolean;
  defaultChecked?: boolean;
  disabled?: boolean;
  onClick: () => void;
  title?: string;
};

export function Toggle({ checked, defaultChecked, disabled, onClick, title }: Props) {
  return (
    <Switch
      checked={checked}
      defaultChecked={defaultChecked}
      className={cn(
        'relative inline-flex h-5 w-10 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent bg-white ring-1 ring-slate-200 transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2',
        disabled && 'opacity-50'
      )}
      disabled={disabled}
      onClick={onClick}
      title={title}
    >
      {({ checked }) => (
        <span
          className={cn(
            checked ? 'translate-x-5 bg-indigo-500' : 'translate-x-0 bg-slate-500',
            'pointer-events-none relative inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out'
          )}
        >
          <span
            className={cn(
              checked ? 'opacity-0 duration-100 ease-out' : 'opacity-100 duration-200 ease-in',
              'absolute inset-0 flex h-full w-full items-center justify-center transition-opacity'
            )}
            aria-hidden="true"
          >
            <span className="h-0.5 w-2.5 rounded-full bg-white" />
          </span>
          <span
            className={cn(
              checked ? 'opacity-100 duration-200 ease-in' : 'opacity-0 duration-100 ease-out',
              'absolute inset-0 flex h-full w-full items-center justify-center transition-opacity'
            )}
            aria-hidden="true"
          >
            <span className="h-1.5 w-1.5 rounded-full bg-white" />
          </span>
        </span>
      )}
    </Switch>
  );
}
