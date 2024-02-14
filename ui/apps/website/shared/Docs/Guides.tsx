import { Button } from "./Button";
import { Heading } from "./Heading";

const guides = [
  {
    href: "/authentication",
    name: "Authentication",
    description: "Learn how to authenticate your API requests.",
  },
  {
    href: "/pagination",
    name: "Pagination",
    description: "Understand how to work with paginated responses.",
  },
  {
    href: "/errors",
    name: "Errors",
    description:
      "Read about the different types of errors returned by the API.",
  },
  {
    href: "/webhooks",
    name: "Webhooks",
    description:
      "Learn how to programmatically configure webhooks for your app.",
  },
];

export function Guide(guide: {
  href: string;
  name: string;
  description: string;
}) {
  return (
    <div key={guide.href}>
      <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
        {guide.name}
      </h3>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
        {guide.description}
      </p>
      <p className="mt-4">
        <Button href={guide.href} variant="text" arrow="right">
          Read more
        </Button>
      </p>
    </div>
  );
}

export function GuideGrid({ cols = 4, children }) {
  return (
    <div
      className={`xl:max-w-none not-prose mt-4 grid grid-cols-1 gap-8 border-t border-slate-900/5 pt-10 dark:border-white/5
      sm:grid-cols-2 xl:grid-cols-${cols}`}
    >
      {children}
    </div>
  );
}

export function Guides() {
  return (
    <div className="my-16 xl:max-w-none">
      <Heading level={2} id="guides">
        Guides
      </Heading>
      <GuideGrid>
        {guides.map((guide) => (
          <Guide {...guide} />
        ))}
      </GuideGrid>
    </div>
  );
}
