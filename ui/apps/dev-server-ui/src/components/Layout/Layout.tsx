import { useEffect, useRef, useState, type ReactNode } from 'react';

import { useBooleanFlag } from '@/hooks/useBooleanFlag';
import LayoutV2 from './LayoutV2';
import SideBar from './SideBar';

export default function Layout({ children }: { children: ReactNode }) {
  const [collapsed, setCollapsed] = useState<boolean | undefined>(undefined);
  // Captured before the persistence effect below writes a value, so V2 can
  // auto-collapse on small viewports when the user has no explicit pref.
  const hasStoredPref = useRef(false);
  const { value: navV2, isReady: navFlagReady } = useBooleanFlag(
    'navigation-v2',
    false,
  );

  useEffect(() => {
    hasStoredPref.current = localStorage.getItem('navCollapsed') !== null;
    setCollapsed(localStorage.getItem('navCollapsed') === 'true');
  }, []);

  useEffect(() => {
    collapsed !== undefined &&
      localStorage.setItem('navCollapsed', JSON.stringify(collapsed));
  }, [collapsed]);

  //
  // don't render until we have the nav collapsed state and the nav version
  // flag to avoid jank / a flash of the wrong navigation
  if (collapsed === undefined || !navFlagReady) {
    return null;
  }

  if (navV2) {
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

  return (
    <div className="no-scrollbar text-basis fixed z-50 flex h-screen w-full flex-row justify-start overflow-y-scroll overscroll-y-none">
      <SideBar collapsed={collapsed} setCollapsed={setCollapsed} />
      <div className="no-scrollbar flex w-full flex-col overflow-x-scroll">
        {children}
      </div>
    </div>
  );
}
