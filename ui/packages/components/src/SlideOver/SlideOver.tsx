'use client';

import { useState } from 'react';
import { classNames } from '@inngest/components/utils/classNames';
import * as Dialog from '@radix-ui/react-dialog';
import { AnimatePresence, motion } from 'framer-motion';

type SlideOverProps = {
  children?: React.ReactNode;
  onClose: () => void;
  size?: 'small' | 'large';
};

export function SlideOver({ children, onClose, size = 'large' }: SlideOverProps) {
  const [isOpen, setOpen] = useState(true);

  function handleClose() {
    setOpen(false);
    // Allows the exit transition to happen before unmounting
    setTimeout(() => {
      onClose();
    }, 500);
  }

  return (
    <Dialog.Root open={isOpen} onOpenChange={handleClose} modal>
      <AnimatePresence>
        {isOpen ? (
          <Dialog.Portal forceMount>
            <Dialog.Overlay asChild>
              <motion.div
                className="fixed inset-0 z-50 bg-white/30 backdrop-blur-[2px] transition-opacity dark:bg-[#04060C]/90"
                aria-hidden="true"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{
                  duration: 0.5,
                  type: 'tween',
                }}
              />
            </Dialog.Overlay>
            {/* Content container */}
            <div className={classNames(size === 'small' ? 'w-2/5' : 'w-4/5', 'fixed inset-0 z-50')}>
              <motion.div
                className="flex h-full w-screen items-center justify-end"
                initial={{ x: '100%' }}
                animate={{ x: 0 }}
                exit={{ x: '100%' }}
                transition={{
                  duration: 0.5,
                  type: 'tween',
                }}
              >
                <Dialog.Content
                  className={classNames(
                    size === 'small' ? 'w-2/5' : 'w-4/5',
                    'bg-slate-910 flex h-full flex-col shadow-xl'
                  )}
                >
                  {children}
                </Dialog.Content>
              </motion.div>
            </div>
          </Dialog.Portal>
        ) : null}
      </AnimatePresence>
    </Dialog.Root>
  );
}
