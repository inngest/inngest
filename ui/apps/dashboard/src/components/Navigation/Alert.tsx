'use client';

import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { RiCloseLine } from '@remixicon/react';

const ALERT_NAME = 'inngest-dismissIAAlert';
//
// TODO: turn this into a proper component
export const Alert = () => {
  const [show, setShow] = useState(false);

  useEffect(() => {
    window.localStorage.getItem(ALERT_NAME) !== 'true' && setShow(true);
  }, []);

  const dismiss = () => {
    setShow(false);
    window.localStorage.setItem(ALERT_NAME, 'true');
  };

  return (
    show && (
      <div className="text-info bg-info border-secondary-2xSubtle mb-5 rounded border py-3 pl-3 pr-2 text-xs leading-tight">
        <div className="gap-x flex flex-row items-start justify-between">
          <div>We&apos;ve reimagined our information architecture for better navigation.</div>
          <Button
            icon={<RiCloseLine className="text-link" />}
            kind="secondary"
            appearance="ghost"
            size="small"
            className="ml-.5"
            onClick={() => dismiss()}
          />
        </div>
        <Link
          href=" https://www.inngest.com/blog/reimagining-information-architecture"
          className="mt-4"
        >
          Read about the redesign
        </Link>
      </div>
    )
  );
};
