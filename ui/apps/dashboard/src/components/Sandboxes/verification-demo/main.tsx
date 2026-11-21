import { createRoot } from 'react-dom/client';
import '@inngest/components/AppRoot/globals.css';
import '@inngest/components/AppRoot/fonts.css';

if (import.meta.env.DEV) {
  const { SandboxesVerificationDemo } = await import(
    './SandboxesVerificationDemo'
  );
  createRoot(document.getElementById('root')!).render(
    <SandboxesVerificationDemo />,
  );
}
