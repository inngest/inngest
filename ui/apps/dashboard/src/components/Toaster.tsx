'use client';

import { Toaster } from 'sonner';

export default function ToasterWrapper() {
  return (
    <Toaster
      toastOptions={{
        // Ensure that the toast is clickable when there are overlays/modals
        className: 'pointer-events-auto drop-shadow-lg',
        style: {
          background: `rgb(var(--color-background-canvas-base))`,
          borderRadius: 0,
          borderWidth: '0px 0px 2px',
          color: `rgb(var(--color-foreground-base))`,
        },
      }}
    />
  );
}
