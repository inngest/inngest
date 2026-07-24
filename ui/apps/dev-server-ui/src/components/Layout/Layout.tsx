import { useEffect, useRef, useState, type ReactNode } from 'react';

import LayoutV2 from './LayoutV2';

export default function Layout({ children }: { children: ReactNode }) {
  const [collapsed, setCollapsed] = useState<boolean | undefined>(undefined);
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
    >
      {children}
    </LayoutV2>
  );
}
