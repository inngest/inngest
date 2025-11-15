import { createFileRoute, Link, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/")({
  component: Home,
  loader: ({ context }) => {
    if (true) {
      redirect({
        to: "/env/$env/apps",
        params: { env: "production" },
        throw: true,
      });
    }
  },
});

function Home() {
  return (
    <div className="px-6 pt-4">
      <Link to="/env/$env/apps" params={{ env: "production" }}>
        Apps &rarr;
      </Link>
    </div>
  );
}
