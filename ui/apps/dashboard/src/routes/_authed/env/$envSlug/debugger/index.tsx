import { Header } from '@inngest/components/Header/Header';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/_authed/env/$envSlug/debugger/')({
  component: DebuggerPage,
});

function DebuggerPage() {
  return (
    <>
      <Header breadcrumb={[{ text: 'Debug' }]} />
      <div>coming soon...</div>
    </>
  );
}
