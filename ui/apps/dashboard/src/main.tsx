import { StrictMode } from 'react';
import { RouterProvider } from '@tanstack/react-router';
import ReactDOM from 'react-dom/client';

import { createTanStackRouter } from './createTanStackRouter';

// Create and render the TanStack Router app
async function initializeApp() {
  try {
    // Create router using shared logic (no basepath for standalone app)
    const router = await createTanStackRouter();

    // Render the app
    const rootElement = document.getElementById('root')!;
    if (!rootElement.innerHTML) {
      const root = ReactDOM.createRoot(rootElement);
      root.render(
        <StrictMode>
          <RouterProvider router={router} />
        </StrictMode>
      );
    }
  } catch (error) {
    console.error('Failed to initialize TanStack Router app:', error);

    // Show error in the root element
    const rootElement = document.getElementById('root');
    if (rootElement) {
      rootElement.innerHTML = `
        <div style="display: flex; justify-content: center; align-items: center; height: 100vh; font-family: sans-serif;">
          <div style="text-align: center; padding: 2rem; border: 1px solid #ef4444; background: #fef2f2; border-radius: 8px;">
            <h2 style="color: #dc2626; margin: 0 0 1rem 0;">Failed to Load TanStack Router</h2>
            <p style="color: #7f1d1d; margin: 0;">${error.message}</p>
          </div>
        </div>
      `;
    }
  }
}

// Initialize the app
initializeApp();
