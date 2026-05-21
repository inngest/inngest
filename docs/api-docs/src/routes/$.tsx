import { Suspense } from "react";
import { createFileRoute, notFound } from "@tanstack/react-router";
import browserCollections from "collections/browser";
import { DocsLayout } from "fumadocs-ui/layouts/docs";
import { MarkdownCopyButton } from "fumadocs-ui/layouts/docs/page";
import {
  DocsBody,
  DocsDescription,
  DocsPage,
  DocsTitle,
} from "fumadocs-ui/page";

import { useMDXComponents } from "@/components/mdx";
import { baseOptions } from "@/lib/layout.shared";
import { getDocPage } from "@/lib/page-data";
import { MarkdownURLCopyButton } from "@/components/MarkdownURLCopyButton";

export const Route = createFileRoute("/$")({
  component: Page,
  loader: async ({ params }) => {
    const slugs =
      (params as { _splat?: string })._splat?.split("/").filter(Boolean) ?? [];
    const data = await getDocPage({ data: { slugs } });
    if (!data) throw notFound();

    await clientLoader.preload(data.path);
    // pageTree is typed as `any` from the server fn (getPageTree() returns ReactNode
    // names per its type, but actual values are plain strings — safe to serialize).
    return data;
  },
});

const clientLoader = browserCollections.docs.createClientLoader<{
  markdownUrl: string;
}>({
  component({ toc, frontmatter, default: MDX }, { markdownUrl }) {
    return (
      <DocsPage toc={toc}>
        <DocsTitle>{frontmatter.title as string}</DocsTitle>
        <DocsDescription className="mb-3">
          {frontmatter.description as string | undefined}
        </DocsDescription>
        <div className="flex flex-row flex-wrap items-center justify-start gap-2 mb-3">
          <MarkdownCopyButton markdownUrl={markdownUrl} />
          <MarkdownURLCopyButton markdownUrl={markdownUrl} />
        </div>
        <DocsBody>
          <MDX components={useMDXComponents()} />
        </DocsBody>
      </DocsPage>
    );
  },
});

function Page() {
  const { path, pageTree, url } = Route.useLoaderData();
  const markdownUrl = toMarkdownUrl(url);

  return (
    <DocsLayout
      {...baseOptions()}
      tree={pageTree}
      githubUrl="https://github.com/inngest/inngest"
    >
      <Suspense>{clientLoader.useContent(path, { markdownUrl })}</Suspense>
    </DocsLayout>
  );
}

function toMarkdownUrl(url: string): string {
  // The .md route resolves an `index.md` slug back to the homepage. Other
  // pages just get a `.md` suffix.
  return url === "/" ? "/index.md" : `${url}.md`;
}
