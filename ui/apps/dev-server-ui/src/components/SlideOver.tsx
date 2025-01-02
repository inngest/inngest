'use client';

import { Fragment, useState } from 'react';
import { Dialog, Transition } from '@headlessui/react';
import { cn } from '@inngest/components/utils/classNames';

type SlideOverProps = {
  children?: React.ReactNode;
  onClose: () => void;
  size?: 'small' | 'large';
};

export default function SlideOver({ children, onClose, size }: SlideOverProps) {
  const [isOpen, setOpen] = useState(true);

  function handleClose() {
    setOpen(false);
    // Allows the leave transition to happen before unmounting
    setTimeout(() => {
      onClose();
    }, 500);
  }

  return (
    <Transition.Root show={isOpen} appear={true} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={handleClose}>
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
              className={cn(
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
                    className={cn(
                      size === 'small' ? 'w-2/5' : 'w-4/5',
                      'bg-canvasBase flex h-full flex-col shadow-xl'
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
