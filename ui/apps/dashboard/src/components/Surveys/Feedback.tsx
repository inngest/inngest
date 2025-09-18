'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button/index';
import { Link } from '@inngest/components/Link';
import { RiCloseLine } from '@remixicon/react';

const HIDE_FEEDBACK = 'inngest-feedback-hide';

export default function NewUser() {
  const [open, setOpen] = useState(() => {
    return typeof window !== 'undefined' && window.localStorage.getItem(HIDE_FEEDBACK) !== 'true';
  });

  const dismiss = () => {
    setOpen(false);
    window.localStorage.setItem(HIDE_FEEDBACK, 'true');
  };

  return (
    open && (
      <div className="bg-canvasBase border-subtle absolute bottom-0 right-0 mb-6 mr-4 w-[430px] rounded border">
        <div className="gap-x flex flex-row items-center justify-between p-3">
          <div className="text-sm leading-tight">We&apos;d love your feedback on Inngest!</div>
          <Button
            icon={<RiCloseLine className="text-subtle h-5 w-5" />}
            kind="secondary"
            appearance="ghost"
            size="small"
            className="ml-.5"
            onClick={() => dismiss()}
          />
        </div>
        <div className="text-muted px-3 pb-3 text-sm">
          The product design team is looking for users interested in talking to us about their
          experience with Inngest. Your feedback will help shape what we build next.
        </div>
        <div className="border-subtle border-t px-3 py-2">
          <Link href="https://forms.gle/JJQrV9gKfetFaiQTA" target="_blank">
            Let&apos;s talk
          </Link>
        </div>
      </div>
    )
  );
}
