'use client';

import { StrictMode, useEffect, useState } from 'react';
import { RouterProvider } from '@tanstack/react-router';

import { createTanStackRouter } from '../../createTanStackRouter';

export default function TanStackRouterApp() {
  const [router, setRouter] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    // Use the shared router creation logic with Next.js basepath
    createTanStackRouter('/tanstack')
      .then(setRouter)
      .catch((err) => {
        console.error('Error creating router:', err);
        console.error('Full error details:', err);
        setError(`Failed to create TanStack Router: ${err.message}`);
      });
  }, []);

  if (error) {
    return (
      <div className="flex h-screen w-full items-center justify-center">
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-center">
          <h2 className="text-lg font-semibold text-red-900">Error Loading Router</h2>
          <p className="text-red-700">{error}</p>
        </div>
      </div>
    );
  }

  if (!router) {
    return (
      <div className="flex h-screen w-full items-center justify-center">
        <div className="rounded-lg bg-blue-50 p-6 text-center">
          <div className="text-lg font-medium text-blue-900">Loading TanStack Router...</div>
          <div className="text-sm text-blue-700">Shared logic with main.tsx</div>
        </div>
      </div>
    );
  }

  return (
    <StrictMode>
      <RouterProvider router={router} />
    </StrictMode>
  );
}
