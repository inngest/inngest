'use client';

import { useEffect, useState } from 'react';
import { useUser } from '@clerk/nextjs';
import { Button } from '@inngest/components/Button/index';
import { Link } from '@inngest/components/Link';
import { isAfter, sub } from '@inngest/components/utils/date';
import { RiCloseLine } from '@remixicon/react';

const HIDE_NEW_USER_SURVEY = 'inngest-new-use-survey-hide';

export default function NewUser() {
  const [open, setOpen] = useState(false);
  const { user } = useUser();

  useEffect(() => {
    if (
      user?.createdAt &&
      isAfter(
        user.createdAt,
        sub(new Date(), {
          months: 2,
        })
      ) &&
      window.localStorage.getItem(HIDE_NEW_USER_SURVEY) !== 'true'
    ) {
      setOpen(true);
    }
  }, [user]);

  const dismiss = () => {
    setOpen(false);
    window.localStorage.setItem(HIDE_NEW_USER_SURVEY, 'true');
  };

  return (
    open && (
      <div className="bg-canvasBase border-subtle absolute bottom-0 right-0 mb-6 mr-4 w-[430px] rounded border">
        <div className="gap-x flex flex-row items-center justify-between p-3">
          <div className="text-sm leading-tight">Got a few minutes?</div>
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
          Inngest&apos;s Product Design would like to hear about your experience onboarding and
          using Inngest. Please fill out this brief 7-question survey on your experience. After
          completion, you will be entered into a drawing for an Amazon gift card.
        </div>
        <div className="border-subtle border-t px-3 py-2">
          <Link href="https://t.maze.co/282304348" arrowOnHover={true} target="_blank">
            Take survey
          </Link>
        </div>
      </div>
    )
  );
}
