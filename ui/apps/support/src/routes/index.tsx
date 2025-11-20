import { createFileRoute, Link, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/")({
  component: Home,
  loader: () => {
    redirect({
      to: "/support",
      throw: true,
    });
  },
});

function Home() {
  return (
    <div className="px-6 pt-4">
      <Link to="/support">Inngest Support Portal &rarr;</Link>
    </div>
  );
}
