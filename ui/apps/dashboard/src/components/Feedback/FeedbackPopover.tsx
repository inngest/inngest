import { Fragment, useState, type ReactNode } from 'react';
import { useOrganization, useUser } from '@clerk/tanstack-react-start';
import { Textarea } from '@inngest/components/Forms/Textarea';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { Button } from '@inngest/components/Button';
import { RiCheckLine } from '@remixicon/react';
import { toast } from 'sonner';

type Props = {
  // Element rendered as the popover trigger (asChild), e.g. a bottom-bar link
  // or a floating icon button. Lets multiple entry points share one dialog.
  trigger: ReactNode;
  align?: 'start' | 'center' | 'end';
  side?: 'top' | 'bottom' | 'left' | 'right';
  // Renders a leading "|" divider before the trigger (used in the bottom bar).
  leadingDivider?: boolean;
};

export default function FeedbackPopover({
  trigger,
  align = 'start',
  side = 'top',
  leadingDivider = false,
}: Props) {
  const { user } = useUser();
  const { organization } = useOrganization();

  const [open, setOpen] = useState(false);
  const [feedback, setFeedback] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [sent, setSent] = useState(false);

  // Cloud-only: user + organization come from Clerk, which is absent in
  // self-hosted/marketplace mode.
  if (!user || !organization) return null;

  const email = user.primaryEmailAddress?.emailAddress;

  // Typing again after a send readies the box for the next submission.
  function onChangeFeedback(value: string) {
    setFeedback(value);
    if (sent) setSent(false);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!feedback.trim() || !user || !organization || !email) return;

    setIsSubmitting(true);

    try {
      const result = await fetch('/api/feedback', {
        method: 'POST',
        credentials: 'include',
        body: JSON.stringify({
          user: { name: user.fullName ?? email, email, clerkId: user.id },
          organization: { name: organization.name, clerkId: organization.id },
          page: window.location.href,
          feedback,
        }),
      });

      if (result.ok) {
        // Keep the popover open with a cleared box so more feedback can be sent.
        setFeedback('');
        setSent(true);
        toast.success('Feedback sent successfully.');
      } else {
        toast.error('Something went wrong. Please try again.');
      }
    } catch {
      toast.error('Something went wrong. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <Fragment>
      {leadingDivider && <span className="text-disabled">|</span>}
      <Popover
        open={open}
        onOpenChange={(next) => {
          setOpen(next);
          if (!next) {
            setFeedback('');
            setSent(false);
          }
        }}
      >
        <PopoverTrigger asChild>{trigger}</PopoverTrigger>
        <PopoverContent align={align} side={side} className="w-[380px] p-4">
          <form onSubmit={onSubmit} className="flex flex-col gap-3">
            <p className="text-basis text-sm font-medium">Feedback</p>
            <Textarea
              value={feedback}
              onChange={onChangeFeedback}
              placeholder="Share your feedback…"
              rows={5}
              required
            />
            {sent && (
              <p className="text-success flex items-center gap-1 text-xs">
                <RiCheckLine className="h-3.5 w-3.5" />
                We've got your feedback. Feel free to send more!
              </p>
            )}
            <div className="flex items-center justify-between gap-3">
              <Button
                type="submit"
                kind="primary"
                label="Submit"
                loading={isSubmitting}
                disabled={!feedback.trim()}
              />
            </div>
          </form>
        </PopoverContent>
      </Popover>
    </Fragment>
  );
}
