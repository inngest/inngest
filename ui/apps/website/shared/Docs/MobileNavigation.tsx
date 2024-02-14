import { createContext, Fragment, useContext, useEffect } from "react";
import { Dialog, Transition } from "@headlessui/react";
import { motion } from "framer-motion";
import create from "zustand";

import { Header } from "./Header";
import { Navigation } from "./Navigation";

function MenuIcon(props) {
  return (
    <svg
      viewBox="0 0 10 9"
      fill="none"
      strokeLinecap="round"
      aria-hidden="true"
      {...props}
    >
      <path d="M.5 1h9M.5 8h9M.5 4.5h9" />
    </svg>
  );
}

function XIcon(props) {
  return (
    <svg
      viewBox="0 0 10 9"
      fill="none"
      strokeLinecap="round"
      aria-hidden="true"
      {...props}
    >
      <path d="m1.5 1 7 7M8.5 1l-7 7" />
    </svg>
  );
}

const IsInsideMobileNavigationContext = createContext(false);

export function useIsInsideMobileNavigation() {
  return useContext(IsInsideMobileNavigationContext);
}

type MobileNavigationStore = {
  isOpen: boolean;
  open: () => void;
  close: () => void;
  toggle: () => void;
};
export const useMobileNavigationStore = create<MobileNavigationStore>(
  (set) => ({
    isOpen: false,
    open: () => set({ isOpen: true }),
    close: () => set({ isOpen: false }),
    toggle: () => set((state) => ({ isOpen: !state.isOpen })),
  })
);

export function MobileNavigation() {
  let isInsideMobileNavigation = useIsInsideMobileNavigation();
  let { isOpen, toggle, close } = useMobileNavigationStore();
  let ToggleIcon = isOpen ? XIcon : MenuIcon;

  // Headless UI doesn't always removing their "portal" dom element,
  // so we remove it manually if needed
  useEffect(() => {
    if (!isInsideMobileNavigation && !isOpen) {
      const portalRoot = document.getElementById("headlessui-portal-root");
      if (portalRoot) {
        portalRoot.remove();
      }
    }
  }, [isInsideMobileNavigation, isOpen]);

  return (
    <IsInsideMobileNavigationContext.Provider value={true}>
      <button
        type="button"
        className="flex h-6 w-6 items-center justify-center rounded-md transition hover:bg-slate-900/5 dark:hover:bg-white/5"
        aria-label="Toggle navigation"
        onClick={toggle}
      >
        <ToggleIcon className="w-2.5 stroke-slate-900 dark:stroke-white" />
      </button>
      {!isInsideMobileNavigation && (
        <Transition.Root show={isOpen} as={Fragment}>
          <Dialog onClose={close} className="fixed inset-0 z-50 lg:hidden">
            <Transition.Child
              as={Fragment}
              enter="duration-300 ease-out"
              enterFrom="opacity-0"
              enterTo="opacity-100"
              leave="duration-200 ease-in"
              leaveFrom="opacity-100"
              leaveTo="opacity-0"
            >
              <div className="fixed inset-0 top-14 bg-slate-400/20 backdrop-blur-sm dark:bg-black/40" />
            </Transition.Child>

            <Dialog.Panel>
              <Transition.Child
                as={Fragment}
                enter="duration-300 ease-out"
                enterFrom="opacity-0"
                enterTo="opacity-100"
                leave="duration-200 ease-in"
                leaveFrom="opacity-100"
                leaveTo="opacity-0"
              >
                <Header />
              </Transition.Child>

              <Transition.Child
                as={Fragment}
                enter="duration-500 ease-in-out"
                enterFrom="-translate-x-full"
                enterTo="translate-x-0"
                leave="duration-500 ease-in-out"
                leaveFrom="translate-x-0"
                leaveTo="-translate-x-full"
              >
                <motion.div
                  layoutScroll
                  className="fixed left-0 top-14 bottom-0 w-full overflow-y-auto bg-white px-4 pt-6 pb-4 shadow-lg shadow-slate-900/10 ring-1 ring-slate-900/7.5 dark:bg-slate-900 dark:ring-slate-800 min-[416px]:max-w-sm sm:px-6 sm:pb-10"
                >
                  <Navigation />
                </motion.div>
              </Transition.Child>
            </Dialog.Panel>
          </Dialog>
        </Transition.Root>
      )}
    </IsInsideMobileNavigationContext.Provider>
  );
}
