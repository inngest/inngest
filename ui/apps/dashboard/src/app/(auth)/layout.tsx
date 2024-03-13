import React from 'react';
import { GoogleAnalytics } from '@next/third-parties/google';

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      {children}
      {process.env.NEXT_PUBLIC_GOOGLE_ANALYTICS_ID && (
        <GoogleAnalytics gaId={process.env.NEXT_PUBLIC_GOOGLE_ANALYTICS_ID} />
      )}
    </>
  );
}
