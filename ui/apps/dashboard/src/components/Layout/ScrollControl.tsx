'use client';

import { useEffect } from 'react';
import { usePathname } from 'next/navigation';

type ScrollControlProps = {
  containerId: string;
};

export default function ScrollControl({ containerId }: ScrollControlProps) {
  const pathname = usePathname();

  useEffect(() => {
    const el = typeof document !== 'undefined' ? document.getElementById(containerId) : null;
    if (!el) return;

    if (pathname === '/env/production/insights') {
      el.style.overflowY = 'hidden';
    } else {
      el.style.overflowY = 'scroll';
    }

    return () => {
      if (!el) return;
      el.style.overflowY = '';
    };
  }, [containerId, pathname]);

  return null;
}
