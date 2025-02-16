import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/env/')({
  component: Component,
});

function Component() {
  return 'Env';
}
