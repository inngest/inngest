import { Outlet, createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/env')({
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <div>
      <div>Route</div>
      <Outlet />
    </div>
  );
}
