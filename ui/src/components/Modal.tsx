import { Fragment } from "react";
import { Dialog, Transition } from "@headlessui/react";

type ModalProps = {
  children?: React.ReactNode;
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
};

export default function Modal({
  children,
  isOpen,
  onClose,
  title,
  description,
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
            className="fixed inset-0 bg-[#04060C]/90 transition-opacity"
            aria-hidden="true"
          />
        </Transition.Child>
        {/* Full-screen container to center the panel */}
        <div className="fixed inset-0 overflow-y-auto">
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
              <Dialog.Panel className="max-w-lg transform overflow-hidden rounded bg-slate-950 shadow-xl transition-all">
                {(title || description) && (
                  <div className="border-b border-slate-800 p-6">
                    <Dialog.Title className="text-white text-xl font-semibold">
                      {title}
                    </Dialog.Title>
                    <Dialog.Description className="text-slate-400 text-sm font-medium">
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
