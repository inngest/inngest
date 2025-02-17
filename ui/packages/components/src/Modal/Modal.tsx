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
  const container = typeof document !== 'undefined' ? document.getElementById('modals') : undefined;
  return (
    <Dialog.Root open={isOpen} onOpenChange={onClose} modal>
      <AnimatePresence>
        <Dialog.Portal container={container}>
          <Dialog.Overlay
            asChild
            className="fixed inset-0 z-[100] backdrop-blur backdrop-invert-[10%] transition-opacity"
            aria-hidden="true"
          >
            {/* Full-screen container to center the panel */}
            <div className="fixed inset-0 z-[100]">
              <motion.div
                className={cn(
                  alignTop ? 'items-baseline' : 'items-center',
                  'flex h-full w-full justify-center p-6'
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
                    'bg-canvasBase shadow-tooltip border-subtle max-h-full overflow-y-auto overflow-x-hidden rounded-md border shadow-2xl'
                  )}
                >
                  {(title || description) && <Header description={description}>{title}</Header>}
                  {children}
                  {footer && <Footer>{footer}</Footer>}
                </Dialog.Content>
              </motion.div>
            </div>
          </Dialog.Overlay>
        </Dialog.Portal>
      </AnimatePresence>
    </Dialog.Root>
  );
}

function Body({ children }: React.PropsWithChildren<{}>) {
  return <div className="text-basis m-6">{children}</div>;
}

function Footer({ children, className }: React.PropsWithChildren<{ className?: string }>) {
  return <div className={cn('border-subtle border-t p-6', className)}>{children}</div>;
}

function Header({
  children,
  description,
}: React.PropsWithChildren<{ description?: React.ReactNode }>) {
  return (
    <div className="bg-canvasBase border-subtle border-b p-6">
      <Dialog.Title className="text-basis text-xl">{children}</Dialog.Title>

      {description && (
        <Dialog.Description className="text-subtle pt-1">{description}</Dialog.Description>
      )}
    </div>
  );
}

Modal.Body = Body;
Modal.Footer = Footer;
Modal.Header = Header;
