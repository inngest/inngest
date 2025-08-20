'use client';

import dynamic from 'next/dynamic';

// Dynamically import the entire TanStack Router app with no SSR
const TanStackRouterApp = dynamic(() => import('./TanStackRouterApp'), {
  ssr: false,
  loading: () => (
    <div className="flex h-screen w-full items-center justify-center">
      <div className="text-lg">Loading TanStack Router...</div>
    </div>
  ),
});

export default function TanStackPage() {
  return (
    <div className="h-screen w-full">
      <TanStackRouterApp />
    </div>
  );
}
