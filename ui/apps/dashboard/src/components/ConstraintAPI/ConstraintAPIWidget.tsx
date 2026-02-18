import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip/Tooltip';
import { RiCloseLine, RiInformationLine } from '@remixicon/react';

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
        description: 'Learn about the Constraint API',
      };
    case 'pending':
      return {
        collapsedText: 'Enrollment pending',
        title: 'Enrollment Pending',
        description: 'Your enrollment is being processed',
      };
    case 'active':
      return {
        collapsedText: 'Constraint API active',
        title: 'Constraint API Active',
        description: 'Your account is using the new infrastructure',
      };
  }
}

export default function ConstraintAPIWidget({
  collapsed,
}: {
  collapsed: boolean;
}) {
  const { isWidgetVisible, constraintAPIData, dismiss } = useConstraintAPI();
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
          className="border-muted text-basis hover:border-basis flex w-full items-center justify-center gap-2 rounded border border-blue-200 bg-blue-50 px-2 py-2.5 text-sm transition-colors"
        >
          <RiInformationLine className="h-[18px] w-[18px] text-blue-600" />
        </button>
      )}

      {!collapsed && (
        <div className="text-basis mb-5 block rounded border border-blue-200 bg-blue-50 p-3 leading-tight">
          <div className="flex min-h-[110px] flex-col justify-between">
            <div>
              <div className="flex items-center justify-between">
                <RiInformationLine className="h-5 w-5 text-blue-600" />
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      icon={<RiCloseLine className="text-subtle" />}
                      kind="secondary"
                      appearance="ghost"
                      size="small"
                      className="hover:bg-blue-100"
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
              <p className="flex items-center gap-1.5 font-medium text-blue-800">
                {content.title}
              </p>
              <p className="text-sm text-blue-700">{content.description}</p>
            </div>
            <Button
              appearance="outlined"
              className="hover:bg-blue-100 w-full text-sm"
              label="Learn More"
              onClick={() => setIsModalOpen(true)}
            />
          </div>
        </div>
      )}

      <ConstraintAPIModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        constraintAPIData={constraintAPIData}
      />
    </>
  );
}
