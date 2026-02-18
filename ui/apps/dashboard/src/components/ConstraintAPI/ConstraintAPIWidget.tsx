import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip/Tooltip';
import { RiCloseLine } from '@remixicon/react';

import { ConstraintAPIModal } from './ConstraintAPIModal';
import { useConstraintAPI } from './useConstraintAPI';
import type { ConstraintAPIData } from './data';

type ContentConfig = {
  collapsedText: string;
  title: string;
  description: string;
};

function getContentForState(
  displayState: ConstraintAPIData['displayState'],
): ContentConfig {
  switch (displayState) {
    case 'not_enrolled':
      return {
        collapsedText: 'Infrastructure upgrade',
        title: 'Infrastructure Upgrade Available',
        description: 'Enroll before February 23, 2026',
      };
    case 'pending':
      return {
        collapsedText: 'Enrollment pending',
        title: 'Enrollment Pending',
        description: 'Enroll before February 23, 2026',
      };
    case 'active':
      return {
        collapsedText: 'Constraint API active',
        title: 'Constraint API Active',
        description: 'Enroll before February 23, 2026',
      };
  }
}

export default function ConstraintAPIWidget({
  collapsed,
}: {
  collapsed: boolean;
}) {
  const { isWidgetVisible, constraintAPIData, dismiss, refetch } =
    useConstraintAPI();
  const [isModalOpen, setIsModalOpen] = useState(false);

  if (!isWidgetVisible || !constraintAPIData) {
    return null;
  }

  const { displayState } = constraintAPIData;
  const content = getContentForState(displayState);

  return (
    <>
      {collapsed && (
        <button
          onClick={() => setIsModalOpen(true)}
          className="flex w-full items-center justify-center rounded border border-amber-200 bg-amber-50 px-2 py-2.5 text-sm transition-colors hover:bg-amber-100"
        />
      )}

      {!collapsed && (
        <div className="text-basis mb-5 block rounded border border-amber-200 bg-amber-50 p-3 leading-tight">
          <div className="flex min-h-[110px] flex-col justify-between">
            <div>
              <div className="flex items-start justify-between">
                <p className="font-medium text-amber-800">{content.title}</p>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      icon={<RiCloseLine className="text-subtle" />}
                      kind="secondary"
                      appearance="ghost"
                      size="small"
                      className="hover:bg-amber-100 -mr-1 -mt-1 shrink-0"
                      onClick={(e) => {
                        e.preventDefault();
                        dismiss();
                      }}
                    />
                  </TooltipTrigger>
                  <TooltipContent side="right" className="max-w-40">
                    <p>Dismiss for 24 hours</p>
                  </TooltipContent>
                </Tooltip>
              </div>
              <p className="text-sm text-amber-700">{content.description}</p>
            </div>
            <button
              className="text-left text-sm text-amber-800 hover:underline"
              onClick={() => setIsModalOpen(true)}
            >
              Learn more
            </button>
          </div>
        </div>
      )}

      <ConstraintAPIModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onEnrolled={refetch}
        constraintAPIData={constraintAPIData}
      />
    </>
  );
}
