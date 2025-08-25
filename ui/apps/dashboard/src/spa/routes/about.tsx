import { createFileRoute } from '@tanstack/react-router';

import TanStackLayout from '../components/TanStackLayout';

function AboutComponent() {
  return (
    <TanStackLayout>
      <div className="p-4">
        <h1 className="mb-4 text-2xl font-bold">About</h1>
        <p className="mb-2">This hybrid implementation demonstrates:</p>
        <ul className="list-inside list-disc space-y-1">
          <li>TanStack Router file-based routing</li>
          <li>Automatic route discovery and type generation</li>
          <li>Integration with existing Next.js application</li>
          <li>Real authentication and data fetching</li>
        </ul>
      </div>
    </TanStackLayout>
  );
}

export const Route = createFileRoute('/about')({
  component: AboutComponent,
});
