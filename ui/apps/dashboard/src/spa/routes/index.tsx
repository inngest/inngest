import { createFileRoute } from '@tanstack/react-router';

import TanStackLayout from '../components/TanStackLayout';

function IndexComponent() {
  return (
    <TanStackLayout>
      <div className="p-4">
        <h1 className="mb-4 text-2xl font-bold">TanStack Router Hybrid SPA</h1>
        <p className="mb-2">This is the TanStack Router SPA running inside Next.js!</p>
        <p className="mb-4">Features:</p>
        <ul className="list-inside list-disc space-y-1">
          <li>File-based routing with auto-generated types</li>
          <li>Clerk authentication integration</li>
          <li>URQL GraphQL client with auth</li>
          <li>React Context for data sharing</li>
          <li>Production-ready hybrid routing</li>
        </ul>
      </div>
    </TanStackLayout>
  );
}

export const Route = createFileRoute('/')({
  component: IndexComponent,
});
