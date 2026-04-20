import { Link, Outlet, createRootRoute } from "@tanstack/react-router";

export const Route = createRootRoute({
  component: Root,
});

function Root() {
  return (
    <>
      <nav>
        <strong>Inngest load-test</strong>
        <Link to="/" activeProps={{ className: "active" }}>
          Configure
        </Link>
        <Link to="/history" activeProps={{ className: "active" }}>
          History
        </Link>
      </nav>
      <Outlet />
    </>
  );
}
