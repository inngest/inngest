import React from 'react';
import { GoogleTagManager } from '@next/third-parties/google';

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      {children}
      {process.env.NEXT_PUBLIC_GTAG_ID && (
        <GoogleTagManager gtmId={process.env.NEXT_PUBLIC_GTAG_ID} />
      )}
    </>
  );
}
