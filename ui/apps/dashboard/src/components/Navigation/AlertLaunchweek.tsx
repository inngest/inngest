import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button/NewButton';
import { Link } from '@inngest/components/Link/NewLink';
import { RiCloseLine } from '@remixicon/react';

const ALERT_NAME = 'inngest-dismissLaunchWeekAlert';
//
// TODO: turn this into a proper component
export const AlertLaunchweek = ({ collapsed }: { collapsed: boolean }) => {
  const [show, setShow] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined') return;

    // Check if already dismissed
    if (window.localStorage.getItem(ALERT_NAME) === 'true') return;

    // Show the alert if not dismissed
    setShow(true);
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
            Launch week is here! Check out our latest features and announcements
            🚀{' '}
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
