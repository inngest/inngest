import { useEffect, useRef, useState } from 'react';
import { useOrganization, useUser } from '@clerk/tanstack-react-start';
import { Button } from '@inngest/components/Button';
import { LabeledCheckbox } from '@inngest/components/Checkbox/Checkbox';
import { Textarea } from '@inngest/components/Forms/Textarea';
import { Modal } from '@inngest/components/Modal';
import { RiCheckLine } from '@remixicon/react';
import { toast } from 'sonner';

import { trackWaitlistFormSubmitted } from '@/utils/analyticsEvents';

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export default function WaitlistModal({ isOpen, onClose }: Props) {
  const { user } = useUser();
  const { organization } = useOrganization();

  const [workflow, setWorkflow] = useState('');
  const [canContact, setCanContact] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  // Guards the join request against React's double-effect (and re-opens) so we
  // don't fire duplicate joins for a single modal session. The server locates
  // the user's row by their Clerk id, so no page id is tracked client-side.
  const joiningRef = useRef(false);

  const email = user?.primaryEmailAddress?.emailAddress;

  function payloadIdentity() {
    if (!user || !organization || !email) return null;
    return {
      user: { name: user.fullName ?? email, email, clerkId: user.id },
      organization: { name: organization.name, clerkId: organization.id },
      page: window.location.href,
    };
  }

  // Record the signup the moment the modal opens (the button click == intent),
  // even if the user later cancels without answering the questions.
  useEffect(() => {
    if (!isOpen || joiningRef.current) return;
    const identity = payloadIdentity();
    if (!identity) return;

    joiningRef.current = true;
    (async () => {
      try {
        const res = await fetch('/api/waitlist', {
          method: 'POST',
          credentials: 'include',
          body: JSON.stringify({ action: 'join', ...identity }),
        });
        // Allow a retry via the Send fallback if the join create failed.
        if (!res.ok) joiningRef.current = false;
      } catch {
        joiningRef.current = false;
      }
    })();
  }, [isOpen]);

  // Cloud-only: Clerk user/org are absent in self-hosted/marketplace mode.
  if (!user || !organization) return null;

  async function onSend() {
    setIsSubmitting(true);
    try {
      const identity = payloadIdentity();
      const res = await fetch('/api/waitlist', {
        method: 'POST',
        credentials: 'include',
        body: JSON.stringify({
          action: 'answers',
          workflow,
          canContact,
          // Included so the server can create the row if the join call failed.
          ...(identity ?? {}),
        }),
      });

      if (res.ok) {
        trackWaitlistFormSubmitted({
          feature: 'sandboxes',
          canContact,
          message: workflow.trim(),
        });
        toast.success("Thanks! We'll be in touch.");
        handleClose();
      } else {
        toast.error('Something went wrong. Please try again.');
      }
    } catch {
      toast.error('Something went wrong. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  }

  function handleClose() {
    setWorkflow('');
    setCanContact(false);
    onClose();
  }

  return (
    <Modal isOpen={isOpen} onClose={handleClose} className="max-w-lg">
      <Modal.Header>
        <span className="flex items-center gap-2">
          <span className="bg-primary-moderate flex h-6 w-6 items-center justify-center rounded-full">
            <RiCheckLine className="text-alwaysWhite h-4 w-4" />
          </span>
          Thank you for joining the waitlist
        </span>
      </Modal.Header>
      <Modal.Body>
        <div className="flex flex-col gap-5">
          <div className="flex flex-col gap-2">
            <label className="text-basis text-sm font-medium">
              Tell us about the workflow you'd want to build using sandboxes?
            </label>
            <Textarea
              value={workflow}
              onChange={setWorkflow}
              placeholder="What are you hoping to build?"
              rows={4}
            />
          </div>
          <LabeledCheckbox
            id="waitlist-can-contact"
            label="Can we contact you to learn more about your use case?"
            checked={canContact}
            onCheckedChange={(checked) => setCanContact(checked === true)}
          />
        </div>
      </Modal.Body>
      <Modal.Footer className="flex justify-end gap-2">
        <Button
          appearance="outlined"
          kind="secondary"
          label="Cancel"
          onClick={handleClose}
        />
        <Button
          kind="primary"
          label="Send"
          loading={isSubmitting}
          onClick={onSend}
        />
      </Modal.Footer>
    </Modal>
  );
}
