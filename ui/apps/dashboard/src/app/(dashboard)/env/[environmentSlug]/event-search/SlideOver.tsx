'use client';

import { Fragment } from 'react';
import { Dialog, Transition } from '@headlessui/react';
import { classNames } from '@inngest/components/utils/classNames';

type Props = {
  children?: React.ReactNode;
  isOpen: boolean;
  onClose: () => void;
  size?: 'small' | 'large';
};

export function SlideOver({ children, isOpen, onClose, size }: Props) {
  return (
    <Transition.Root show={isOpen} appear={true} as={Fragment}>
      <Dialog as="div" className="relative z-10" onClose={onClose}>
        <Transition.Child
          as={Fragment}
          appear={true}
          enter="ease-in-out duration-500"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in-out duration-500"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 z-10 bg-[#04060C]/90 transition-opacity" />
        </Transition.Child>
        <div className="fixed inset-0 z-10 overflow-hidden">
          <div className="absolute inset-0 overflow-hidden">
            <div
              className={classNames(
                size === 'small' ? 'w-2/5' : 'w-4/5',
                'pointer-events-none fixed inset-y-0 right-0 flex '
              )}
            >
              <Transition.Child
                as="div"
                enter="transform transition ease-in-out duration-500 sm:duration-700"
                enterFrom="translate-x-full"
                enterTo="translate-x-0"
                leave="transform transition ease-in-out duration-500 sm:duration-700"
                leaveFrom="translate-x-0"
                leaveTo="translate-x-full"
              >
                <Dialog.Panel className="pointer-events-auto relative h-full w-screen">
                  <Transition.Child
                    as="div"
                    enter="ease-in-out duration-500"
                    enterFrom="opacity-0"
                    enterTo="opacity-100"
                    leave="ease-in-out duration-500"
                    leaveFrom="opacity-100"
                    leaveTo="opacity-0"
                  />
                  <div
                    className={classNames(
                      size === 'small' ? 'w-2/5' : 'w-4/5',
                      'bg-slate-940 flex h-full flex-col shadow-xl'
                    )}
                  >
                    {children}
                  </div>
                </Dialog.Panel>
              </Transition.Child>
            </div>
          </div>
        </div>
      </Dialog>
    </Transition.Root>
  );
}
