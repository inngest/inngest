'use client';

import { useState } from 'react';
import Image from 'next/image';
import { Button } from '@inngest/components/Button/index';
import { RiCloseLine } from '@remixicon/react';

import darkModeDark from '@/images/dark-mode-dark.png';
import darkModeLight from '@/images/dark-mode-light.png';

const HIDE_DARK_MODE_POPOVER = 'inngest-dark-mode-popover-hide';

export default function DarkModePopover() {
  const [open, setOpen] = useState(() => {
    return (
      typeof window !== 'undefined' &&
      window.localStorage.getItem(HIDE_DARK_MODE_POPOVER) !== 'true'
    );
  });

  const dismiss = () => {
    setOpen(false);
    window.localStorage.setItem(HIDE_DARK_MODE_POPOVER, 'true');
  };

  return (
    open && (
      <div className="bg-canvasBase border-subtle absolute bottom-0 right-0 mb-6 mr-4 w-[400px] overflow-hidden rounded-lg border md:h-[200px]">
        <div className="p-3">
          <div className="mb-1 flex justify-between">
            <p className="mb-1 mt-6 text-3xl">Introducing dark mode</p>
            <Button
              icon={<RiCloseLine />}
              kind="secondary"
              appearance="ghost"
              size="small"
              onClick={() => dismiss()}
            />
          </div>
          <p className="text-muted mb-3 text-sm md:w-1/2">
            We&apos;re excited to release this highly requested feature! Change your theme in the
            settings dropdown.
          </p>
          <Image
            src={darkModeLight}
            alt="screenshot of dark mode toggle"
            className="absolute -bottom-1 -right-[1px] mt-1 hidden w-1/2 md:block dark:md:hidden"
          />
          <Image
            src={darkModeDark}
            alt="screenshot of dark mode toggle"
            className="absolute -bottom-1 -right-[1px] hidden w-1/2 dark:md:block"
          />
        </div>
      </div>
    )
  );
}
