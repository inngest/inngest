import { createFileRoute, Link, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/")({
  component: Home,
  loader: () => {
    redirect({
      to: "/env/$envSlug/apps",
      params: { envSlug: "production" },
      throw: true,
    });
  },
});

function Home() {
  return (
    <div className="px-6 pt-4">
      <Link to="/env/$envSlug/apps" params={{ envSlug: "production" }}>
        Apps &rarr;
      </Link>
    </div>
  );
}
