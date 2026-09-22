import { createFileRoute, redirect } from '@tanstack/react-router';
import { isHosted } from '@/utils/devServer';

export const Route = createFileRoute('/')({
  component: Home,
  head: () =>
    isHosted
      ? {
          meta: [{ name: 'robots', content: 'index, follow' }],
          links: [{ rel: 'canonical', href: 'https://www.inngest.com/dev' }],
        }
      : {},
  loader: () => {
    if (isHosted) return;
    redirect({
      to: '/runs',
      throw: true,
    });
  },
});

function Home() {
  if (isHosted) return null;
  return (
    <div className="flex flex-col items-center justify-start mt-6 gap-2">
      <h1>Index Route</h1>
      dev server coming soon...
    </div>
  );
}
