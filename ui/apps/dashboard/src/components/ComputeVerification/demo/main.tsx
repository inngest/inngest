import { createRoot } from 'react-dom/client';
import '@inngest/components/AppRoot/globals.css';
import '@inngest/components/AppRoot/fonts.css';

// This entry is served only by vite.verification.config.ts in development.
if (import.meta.env.DEV) {
  const { VerificationDemo } = await import('./VerificationDemo');
  createRoot(document.getElementById('root')!).render(<VerificationDemo />);
}
