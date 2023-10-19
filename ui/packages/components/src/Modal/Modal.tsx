import { classNames } from '@inngest/components/utils/classNames';
import * as Dialog from '@radix-ui/react-dialog';
import { AnimatePresence, motion } from 'framer-motion';

type ModalProps = {
  children?: React.ReactNode;
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  className?: string;
};

export function Modal({
  children,
  isOpen,
  onClose,
  title,
  description,
  className = 'max-w-lg',
}: ModalProps) {
  return (
    <Dialog.Root open={isOpen} onOpenChange={onClose} modal>
      <AnimatePresence>
        <Dialog.Portal>
          <Dialog.Overlay asChild>
            <div
              className="fixed inset-0 z-[100] bg-[#04060C]/90 transition-opacity"
              aria-hidden="true"
            />
          </Dialog.Overlay>
          {/* Full-screen container to center the panel */}
          <div className="fixed inset-0 z-[100] overflow-y-auto">
            <div className="flex min-h-full items-center justify-center p-6">
              <motion.div
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
                <Dialog.Content
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
                </Dialog.Content>
              </motion.div>
            </div>
          </div>
        </Dialog.Portal>
      </AnimatePresence>
    </Dialog.Root>
  );
}
