'use client';

import { Fragment } from 'react';
import { Menu, Transition } from '@headlessui/react';
import { ChevronDownIcon } from '@heroicons/react/20/solid';

import cn from '@/utils/cn';

const contextButtonStyles = {
  light:
    'shadow-outline-secondary-light bg-slate-50 text-slate-700 hover:bg-slate-100 rounded-[6px]',
  dark: 'shadow-outline-secondary-dark bg-slate-800 text-white  text-shadow hover:bg-slate-700 rounded-[6px]',
  nav: 'text-white hover:bg-slate-800 pl-4 pr-6 h-full border-l border-slate-800',
};

const contextSheetStyles = {
  light: 'shadow-outline-secondary-light bg-slate-50',
  dark: 'shadow-outline-secondary-dark bg-slate-900',
  nav: 'bg-slate-940/95 backdrop-blur divide-y divide-dashed divide-slate-700',
};

type DropdownProps = {
  label: React.ReactNode;
  context?: 'dark' | 'light' | 'nav';
  children?: React.ReactNode;
};

export default function Dropdown({ label, children, context = 'dark' }: DropdownProps) {
  return (
    <Menu
      as="div"
      className={cn(`relative inline-block text-left`, context === 'nav' ? ' self-stretch' : '')}
    >
      <Menu.Button
        className={cn(
          'font-regular flex items-center gap-1.5 py-1.5 pl-3 pr-3 text-sm tracking-wide transition-all',
          contextButtonStyles[context]
        )}
      >
        {label}
        <ChevronDownIcon className="h-4 w-4 text-slate-500" aria-hidden="true" />
      </Menu.Button>

      <Transition
        as={Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <Menu.Items
          className={cn(
            'absolute right-1 z-10 mt-1 flex min-w-[200px] origin-top-right flex-col items-stretch rounded-md text-sm focus:outline-none',
            contextSheetStyles[context]
          )}
        >
          {children}
        </Menu.Items>
      </Transition>
    </Menu>
  );
}
