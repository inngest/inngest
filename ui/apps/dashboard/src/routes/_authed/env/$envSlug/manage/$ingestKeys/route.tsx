import Keys from '@/components/Manage/Keys';
import { createFileRoute, Outlet } from '@tanstack/react-router';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/manage/$ingestKeys',
)({
  component: KeysLayout,
});

function KeysLayout() {
  return (
    <div className="flex min-h-0 flex-1">
      <div className="border-muted w-80 flex-shrink-0 border-r">
        <Keys />
      </div>
      <div className="text-basis h-full min-w-0 flex-1 overflow-y-auto">
        <Outlet />
      </div>
    </div>
  );
}
