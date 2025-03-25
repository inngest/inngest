'use client';

import { useEffect, useState } from 'react';
import { cn } from '@inngest/components/utils/classNames';
import * as Dialog from '@radix-ui/react-dialog';
import { AnimatePresence, motion } from 'framer-motion';

type SlideOverProps = {
  children?: React.ReactNode;
  onClose: () => void;
  size?: 'small' | 'large';
};

export function SlideOver({ children, onClose, size = 'large' }: SlideOverProps) {
  // This hack is needed to prevent hydration errors.
  // The Radix Dialog is not rendered correctly server side, so we need to prevent it from rendering until the client side hydration is complete (and `useEffect` is run).
  // The issue is reported here: https://github.com/radix-ui/primitives/issues/1386
  const [isOpen, setOpen] = useState(false);

  useEffect(() => {
    setOpen(true);
  }, []);

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
                className="fixed inset-0 z-50 bg-black/50 backdrop-blur-[2px] transition-opacity dark:bg-[#04060C]/90"
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
            <div className={cn(size === 'small' ? 'w-2/5' : 'w-4/5', 'fixed inset-0 z-50')}>
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
                  onOpenAutoFocus={(event: Event) => event.preventDefault()}
                  className={cn(
                    size === 'small' ? 'w-2/5' : 'w-4/5',
                    'bg-canvasBase flex h-full flex-col shadow-xl'
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
