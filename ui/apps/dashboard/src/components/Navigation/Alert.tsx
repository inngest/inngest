'use client';

import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { RiCloseLine } from '@remixicon/react';

const ALERT_NAME = 'inngest-dismissLaunchWeekAlert';
//
// TODO: turn this into a proper component
export const Alert = ({ collapsed }: { collapsed: boolean }) => {
  const [show, setShow] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined') return;

    // Check if already dismissed
    if (window.localStorage.getItem(ALERT_NAME) === 'true') return;

    // Check if today is during launch week (September 17-21, 2025)
    const today = new Date();
    const startDate = new Date(2025, 8, 17); // September 17th, 2025
    const endDate = new Date(2025, 8, 21); // September 21st, 2025

    // Check if today is within the launch week range
    const isDuringLaunchWeek = today >= startDate && today <= endDate;

    if (isDuringLaunchWeek) {
      setShow(true);
    }
  }, []);

  const dismiss = () => {
    setShow(false);
    window.localStorage.setItem(ALERT_NAME, 'true');
  };

  return (
    show &&
    !collapsed && (
      <div className="text-basis bg-info border-secondary-2xSubtle mb-5 rounded border py-3 pl-3 pr-2 text-xs leading-tight">
        <div className="gap-x flex flex-row items-start justify-between">
          <div className="pt-1">
            Launch week is here! Check out our latest features and announcements 🚀{' '}
          </div>
          <Button
            icon={<RiCloseLine className="text-link" />}
            kind="secondary"
            appearance="ghost"
            size="small"
            className="ml-.5 self-start"
            onClick={() => dismiss()}
          />
        </div>
        <Link href="https://www.inngest.com/blog" className="mt-4 text-xs">
          View announcements
        </Link>
      </div>
    )
  );
};
