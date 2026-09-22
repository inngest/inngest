import { useEffect, useRef, useState, type ReactNode } from 'react';

import LayoutV2 from './LayoutV2';
import { isHosted } from '@/utils/devServer';

export default function Layout({
  children,
  connected = true,
}: {
  children: ReactNode;
  connected?: boolean;
}) {
  // the hosted /dev page needs the dashboard shell in its HTML.  an empty
  // page gives search engines no setup instructions before JavaScript runs.
  const [collapsed, setCollapsed] = useState<boolean | undefined>(
    isHosted ? false : undefined,
  );
  // Captured before the persistence effect below writes a value, so V2 can
  // auto-collapse on small viewports when the user has no explicit pref.
  const hasStoredPref = useRef(false);

  useEffect(() => {
    hasStoredPref.current = localStorage.getItem('navCollapsed') !== null;
    setCollapsed(localStorage.getItem('navCollapsed') === 'true');
  }, []);

  useEffect(() => {
    collapsed !== undefined &&
      localStorage.setItem('navCollapsed', JSON.stringify(collapsed));
  }, [collapsed]);

  //
  // don't render until we have the nav collapsed state to avoid jank / a flash
  // of the wrong collapsed state
  if (collapsed === undefined) {
    return null;
  }

  return (
    <LayoutV2
      collapsed={collapsed}
      setCollapsed={setCollapsed}
      hasStoredPref={hasStoredPref.current}
      connected={connected}
    >
      {children}
    </LayoutV2>
  );
}
