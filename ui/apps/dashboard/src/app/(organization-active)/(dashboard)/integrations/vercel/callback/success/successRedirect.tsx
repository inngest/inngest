'use client';

import { NewButton } from '@inngest/components/Button/index';

type SuccessProps = {
  searchParams: {
    onSuccessRedirectURL: string;
  };
};

export default function SuccessRedirect({ searchParams }: SuccessProps) {
  const popout = typeof window !== 'undefined' && window.opener && window.opener !== window;

  return (
    <NewButton
      kind="primary"
      appearance="solid"
      size="medium"
      label="Continue to Inngest Vercel Dashbaord"
      href={popout ? searchParams.onSuccessRedirectURL : '/settings/integrations/vercel'}
    />
  );
}
