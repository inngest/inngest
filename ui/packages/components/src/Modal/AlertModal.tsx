import { cn } from '@inngest/components/utils/classNames';
import * as AlertDialog from '@radix-ui/react-alert-dialog';
import { AnimatePresence, motion } from 'framer-motion';

import { Button } from '../Button';

type AlertModalProps = {
  children?: React.ReactNode;
  isLoading?: boolean;
  isOpen: boolean;
  onClose: () => void;
  title?: string | React.ReactNode;
  description?: string;
  className?: string;
  onSubmit: () => void | Promise<void>;
};

export function AlertModal({
  children,
  isLoading = false,
  isOpen,
  onClose,
  onSubmit,
  title = 'Are you sure you want to delete?',
  description,
  className = 'w-1/4',
}: AlertModalProps) {
  const container = document.getElementById('modals');
  return (
    <AlertDialog.Root open={isOpen} onOpenChange={onClose}>
      <AnimatePresence>
        <AlertDialog.Portal container={container}>
          <AlertDialog.Overlay asChild>
            <div
              className="fixed inset-0 z-[100] bg-black/50 backdrop-blur-[2px] transition-opacity dark:bg-[#04060C]/90"
              aria-hidden="true"
            />
          </AlertDialog.Overlay>
          {/* Full-screen container to center the panel */}
          <div className="fixed inset-0 z-[100] overflow-y-auto">
            <motion.div
              className="flex min-h-full w-full items-center justify-center p-6"
              initial={{ y: -20, opacity: 0.2 }}
              animate={{ y: 0, opacity: 1 }}
              exit={{
                y: -20,
                opacity: 0.2,
                transition: { duration: 0.2, type: 'tween' },
              }}
              transition={{
                duration: 0.15,
                type: 'tween',
              }}
            >
              <AlertDialog.Content
                className={cn(
                  className,
                  'dark:bg-slate-910 transform overflow-hidden rounded-lg bg-white shadow-xl transition-all'
                )}
              >
                {(title || description) && (
                  <div className="dark:bg-slate-910 border-b border-slate-200 bg-slate-900 p-6 dark:border-slate-800">
                    <AlertDialog.Title className="dark:bg-slate-910 bg-slate-900 text-xl font-semibold text-white">
                      {title}
                    </AlertDialog.Title>
                    <AlertDialog.Description className="text-sm text-slate-500 dark:font-medium dark:text-slate-400">
                      {description}
                    </AlertDialog.Description>
                  </div>
                )}
                {children}
                <div className="flex justify-end gap-2 p-6 dark:border-slate-800">
                  <AlertDialog.Cancel asChild>
                    <Button appearance="outlined" disabled={isLoading} label="No" />
                  </AlertDialog.Cancel>
                  <Button
                    disabled={isLoading}
                    kind="danger"
                    label="Yes"
                    loading={isLoading}
                    btnAction={async () => {
                      try {
                        await onSubmit();
                        onClose();
                      } catch {}
                    }}
                  />
                </div>
              </AlertDialog.Content>
            </motion.div>
          </div>
        </AlertDialog.Portal>
      </AnimatePresence>
    </AlertDialog.Root>
  );
}
