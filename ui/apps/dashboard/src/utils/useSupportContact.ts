'use client';

import { useCallback } from 'react';
import { useUser } from '@clerk/nextjs';
import { useBooleanLocalStorage } from '@inngest/components/hooks/useBooleanLocalStorage';
import { toast } from 'sonner';

export function useSupportContact() {
  const hasContacted = useBooleanLocalStorage('support:contacted', false);
  const { user } = useUser();

  const contactSupport = useCallback(() => {
    try {
      if (hasContacted.value) {
        return;
      }

      window.inngest.send({
        name: 'support/slack.channel.requested',
        data: {
          email: user?.emailAddresses[0]?.emailAddress ?? '',
        },
      });

      hasContacted.set(true);

      toast.success("Thanks for contacting us, you'll hear from us within 24 hours!", {
        duration: 4000,
      });
    } catch (error) {
      hasContacted.set(false);
      console.error('Failed to contact support:', error);
      toast.error('Something went wrong. Please try again.');
    }
  }, [hasContacted, user]);

  return {
    contactSupport,
    hasContactedSupport: hasContacted.value,
    isReady: hasContacted.isReady,
  };
}
