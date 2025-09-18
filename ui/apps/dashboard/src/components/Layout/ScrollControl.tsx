'use client';

import { useEffect } from 'react';
import { usePathname } from 'next/navigation';

type ScrollControlProps = {
  containerId: string;
};

export default function ScrollControl({ containerId }: ScrollControlProps) {
  const pathname = usePathname();

  useEffect(() => {
    const el = document.getElementById(containerId);
    if (!el) return;

    el.style.overflowY = pathname === '/env/production/insights' ? 'hidden' : 'scroll';

    return () => {
      el.style.overflowY = '';
    };
  }, [containerId, pathname]);

  return null;
}
