import { cn } from '@inngest/components/utils/classNames';
import * as Dialog from '@radix-ui/react-dialog';
import { AnimatePresence, motion } from 'framer-motion';

type ModalProps = {
  children?: React.ReactNode;
  isOpen: boolean;
  onClose: (open: boolean) => void;

  /** @deprecated Use Modal.Header instead. */
  title?: string | React.ReactNode;

  /** @deprecated Use description prop in Modal.Header instead. */
  description?: string;
  className?: string;

  /** @deprecated Use Modal.Footer instead. */
  footer?: React.ReactNode;

  alignTop?: boolean;
};

export function Modal({
  children,
  isOpen,
  onClose,
  title,
  description,
  className = 'max-w-lg',
  footer,
  alignTop,
}: ModalProps) {
  return (
    <Dialog.Root open={isOpen} onOpenChange={onClose} modal>
      <AnimatePresence>
        <Dialog.Portal>
          <Dialog.Overlay asChild>
            <div
              className="fixed inset-0 z-50 bg-black/50 backdrop-blur-[2px] transition-opacity dark:bg-[#04060C]/90"
              aria-hidden="true"
            />
          </Dialog.Overlay>
          {/* Full-screen container to center the panel */}
          <div className="fixed inset-0 z-50 overflow-y-auto">
            <motion.div
              className={cn(
                alignTop ? 'items-baseline' : 'items-center',
                'flex min-h-full w-full justify-center p-6'
              )}
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
                className={cn(
                  className,
                  'dark:bg-slate-910 transform overflow-hidden rounded-lg bg-white shadow-xl transition-all'
                )}
              >
                {(title || description) && <Header description={description}>{title}</Header>}
                {children}
                {footer && <Footer>{footer}</Footer>}
              </Dialog.Content>
            </motion.div>
          </div>
        </Dialog.Portal>
      </AnimatePresence>
    </Dialog.Root>
  );
}

function Body({ children }: React.PropsWithChildren<{}>) {
  return <div className="m-6">{children}</div>;
}

function Footer({ children, className }: React.PropsWithChildren<{ className?: string }>) {
  return (
    <div className={cn('border-t border-slate-200 p-6 dark:border-slate-800', className)}>
      {children}
    </div>
  );
}

function Header({
  children,
  description,
}: React.PropsWithChildren<{ description?: React.ReactNode }>) {
  return (
    <div className="dark:bg-slate-910 border-b border-slate-200 bg-slate-900 p-6 dark:border-slate-800">
      <Dialog.Title className="dark:bg-slate-910 bg-slate-900 text-xl font-semibold text-white">
        {children}
      </Dialog.Title>

      {description && (
        <Dialog.Description className="dark:bg-slate-910 mt-2 bg-slate-900 text-indigo-100 dark:font-medium dark:text-slate-400">
          {description}
        </Dialog.Description>
      )}
    </div>
  );
}

Modal.Body = Body;
Modal.Footer = Footer;
Modal.Header = Header;
