import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/')({
  component: Component,
});

function Component() {
  return <div className="bg-red-50 text-red-500">Home</div>;
}
