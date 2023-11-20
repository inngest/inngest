'use client';

import { SignOutButton as ClerkSignOutButton } from '@clerk/nextjs';

type SignOutButtonProps = {
  children: React.ReactNode;
  className?: string;
};

export default function SignOutButton({ children, className }: SignOutButtonProps) {
  return (
    <ClerkSignOutButton
      // @ts-expect-error type is wrong. className is accepted as a prop.
      className={className}
      signOutCallback={() => {
        window.location.href = process.env.NEXT_PUBLIC_SIGN_IN_PATH || '/sign-in';
      }}
    >
      {children}
    </ClerkSignOutButton>
  );
}
