import { cn } from '@inngest/components/utils/classNames';
import * as AlertDialog from '@radix-ui/react-alert-dialog';
import { AnimatePresence, motion } from 'framer-motion';

import { Button, type ButtonKind } from '../Button';

type AlertModalProps = {
  children?: React.ReactNode;
  isLoading?: boolean;
  isOpen: boolean;
  onClose: () => void;
  title?: string | React.ReactNode;
  description?: string;
  className?: string;
  onSubmit: () => void | Promise<void>;
  confirmButtonLabel?: string | React.ReactNode;
  cancelButtonLabel?: string | React.ReactNode;
  confirmButtonKind?: ButtonKind;
  autoClose?: boolean;
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
  confirmButtonLabel = 'Yes',
  cancelButtonLabel = 'No',
  confirmButtonKind = 'danger',
  autoClose = true,
}: AlertModalProps) {
  let container = null;
  if (globalThis.document) {
    container = document.getElementById('modals');
  }

  return (
    <AlertDialog.Root open={isOpen} onOpenChange={onClose}>
      <AnimatePresence>
        <AlertDialog.Portal container={container}>
          <AlertDialog.Overlay asChild>
            <div
              className="bg-overlay/20 dark:bg-overlay/50 fixed inset-0 z-[100] transition-opacity"
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
                  'bg-modalBase text-basis transform overflow-hidden rounded-md shadow-xl transition-all'
                )}
              >
                {(title || description) && (
                  <div className="border-subtle bg-modalBase border-b p-6">
                    <AlertDialog.Title className="text-basis text-xl font-semibold">
                      {title}
                    </AlertDialog.Title>
                    <AlertDialog.Description className="text-subtle text-sm">
                      {description}
                    </AlertDialog.Description>
                  </div>
                )}
                {children}
                <div className="flex justify-end gap-2 p-6">
                  <AlertDialog.Cancel asChild>
                    <Button
                      appearance="outlined"
                      kind="secondary"
                      disabled={isLoading}
                      label={cancelButtonLabel}
                    />
                  </AlertDialog.Cancel>
                  <Button
                    disabled={isLoading}
                    kind={confirmButtonKind}
                    label={confirmButtonLabel}
                    loading={isLoading}
                    onClick={async () => {
                      try {
                        await onSubmit();
                        autoClose && onClose();
                      } catch (e) {
                        console.error('error submiting alert modal', e);
                      }
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
