import { Fragment } from 'react';
import { Dialog, Transition } from '@headlessui/react';
import { classNames } from '@inngest/components/utils/classNames';

type ModalProps = {
  children?: React.ReactNode;
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  className?: string;
};

export default function Modal({
  children,
  isOpen,
  onClose,
  title,
  description,
  className = 'max-w-lg',
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
            className="fixed inset-0 z-50 bg-[#04060C]/90 transition-opacity"
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
                className={classNames(
                  className,
                  'bg-slate-910 transform overflow-hidden rounded-lg shadow-xl transition-all'
                )}
              >
                {(title || description) && (
                  <div className="border-b border-slate-800 p-6">
                    <Dialog.Title className="text-xl font-semibold text-white">
                      {title}
                    </Dialog.Title>
                    <Dialog.Description className="text-sm font-medium text-slate-400">
                      {description}
                    </Dialog.Description>
                  </div>
                )}
                {children}
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}
