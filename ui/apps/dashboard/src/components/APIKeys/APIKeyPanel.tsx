import { useRef, type ReactNode } from 'react';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import { SlideOver } from '@inngest/components/SlideOver';
import { RiCloseLine } from '@remixicon/react';

export function APIKeyPanel({
  title,
  children,
  onClose,
  saving = false,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
  saving?: boolean;
}) {
  const closeButton = useRef<HTMLButtonElement>(null);

  return (
    <SlideOver
      onClose={onClose}
      size="fixed-500"
      isDismissible={!saving}
      initialFocus={closeButton}
    >
      <div className="flex h-full w-screen max-w-[500px] flex-col">
        <Modal.Header>
          <span className="flex items-center justify-between gap-4 text-sm font-medium">
            {title}
            <Button
              ref={closeButton}
              kind="secondary"
              appearance="ghost"
              size="small"
              icon={<RiCloseLine />}
              aria-label="Close API key panel"
              disabled={saving}
              onClick={onClose}
            />
          </span>
        </Modal.Header>
        <div className="min-h-0 flex-1 overflow-y-auto p-6">{children}</div>
      </div>
    </SlideOver>
  );
}
