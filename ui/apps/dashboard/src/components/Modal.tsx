import { Fragment, useState } from 'react';
import { Dialog, Transition } from '@headlessui/react';

import cn from '@/utils/cn';

interface ModalProps {
  children?: React.ReactNode;
  backdropClassName?: string;
  className?: string;
  isOpen: boolean;
  onClose: () => void;
}

export default function Modal({
  children,
  className = '',
  backdropClassName = '',
  isOpen,
  onClose,
}: ModalProps) {
  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog onClose={onClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          {/* The backdrop, rendered as a fixed sibling to the panel container */}
          <div
            className={cn('fixed inset-0 z-50 bg-black/50 transition-opacity', backdropClassName)}
            aria-hidden="true"
          />
        </Transition.Child>
        {/* Full-screen container to center the panel */}
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-6">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              {/* The actual dialog panel  */}
              <Dialog.Panel
                className={cn(
                  'max-w-md transform overflow-hidden rounded bg-white p-4 shadow-xl transition-all',
                  className
                )}
              >
                {children}
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}
