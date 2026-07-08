import { Button } from '@inngest/components/Button';
import { RiQuestionAnswerLine } from '@remixicon/react';

import FeedbackPopover from './FeedbackPopover';

// A fixed, icon-only primary button anchored to the bottom-right corner that
// opens the shared feedback dialog. Reuses FeedbackPopover so submission
// behaves identically to the bottom-bar entry point.
export default function FeedbackFloatingButton() {
  return (
    <FeedbackPopover
      align="end"
      side="top"
      trigger={
        <Button
          kind="primary"
          size="large"
          icon={<RiQuestionAnswerLine />}
          aria-label="Send feedback"
          className="fixed bottom-10 right-8 z-50 shadow-lg"
        />
      }
    />
  );
}
