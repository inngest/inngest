import { classNames } from '@inngest/components/utils/classNames';
import * as AlertDialog from '@radix-ui/react-alert-dialog';
import { AnimatePresence, motion } from 'framer-motion';

import { Button } from '../Button';

type AlertModalProps = {
  children?: React.ReactNode;
  isOpen: boolean;
  onClose: () => void;
  title?: string | React.ReactNode;
  description?: string;
  className?: string;
  primaryAction: {
    label: string;
    btnAction: () => void;
  };
};

export function AlertModal({
  children,
  isOpen,
  onClose,
  primaryAction,
  title = 'Are you sure you want to delete?',
  description,
  className = 'w-1/4',
}: AlertModalProps) {
  return (
    <AlertDialog.Root open={isOpen} onOpenChange={onClose}>
      <AnimatePresence>
        <AlertDialog.Portal>
          <AlertDialog.Overlay asChild>
            <div
              className="fixed inset-0 z-50 bg-white/30 backdrop-blur-[2px] transition-opacity dark:bg-[#04060C]/90"
              aria-hidden="true"
            />
          </AlertDialog.Overlay>
          {/* Full-screen container to center the panel */}
          <div className="fixed inset-0 z-50 overflow-y-auto">
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
                className={classNames(
                  className,
                  'dark:bg-slate-910 transform overflow-hidden rounded-lg bg-white shadow-xl transition-all'
                )}
              >
                {(title || description) && (
                  <div className="p-6">
                    <AlertDialog.Title className="text-xl font-semibold text-slate-600 dark:text-white">
                      {title}
                    </AlertDialog.Title>
                    <AlertDialog.Description className="text-sm text-slate-500 dark:font-medium dark:text-slate-400">
                      {description}
                    </AlertDialog.Description>
                    {children}
                  </div>
                )}
                <div className="flex justify-end gap-2 px-6 pb-6 dark:border-slate-800">
                  <AlertDialog.Cancel asChild>
                    <Button appearance="outlined" label="Cancel" />
                  </AlertDialog.Cancel>
                  <Button
                    kind="danger"
                    label={primaryAction.label}
                    btnAction={primaryAction.btnAction}
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
