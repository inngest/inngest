import { Suspense } from 'react';
import { createFileRoute, notFound } from '@tanstack/react-router';
import browserCollections from 'collections/browser';
import { DocsLayout } from 'fumadocs-ui/layouts/docs';
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from 'fumadocs-ui/page';

import { useMDXComponents } from '@/components/mdx';
import { baseOptions } from '@/lib/layout.shared';
import { getDocPage } from '@/lib/page-data';

export const Route = createFileRoute('/$')({
  component: Page,
  loader: async ({ params }) => {
    const slugs = (params as { _splat?: string })._splat?.split('/').filter(Boolean) ?? [];
    const data = await getDocPage({ data: { slugs } });
    if (!data) throw notFound();

    await clientLoader.preload(data.path);
    // pageTree is typed as `any` from the server fn (getPageTree() returns ReactNode
    // names per its type, but actual values are plain strings — safe to serialize).
    return data;
  },
});

const clientLoader = browserCollections.docs.createClientLoader({
  component({ toc, frontmatter, default: MDX }, _props: undefined) {
    return (
      <DocsPage toc={toc}>
        <DocsTitle>{frontmatter.title as string}</DocsTitle>
        <DocsDescription>{frontmatter.description as string | undefined}</DocsDescription>
        <DocsBody>
          <MDX components={useMDXComponents()} />
        </DocsBody>
      </DocsPage>
    );
  },
});

function Page() {
  const { path, pageTree } = Route.useLoaderData();

  return (
    <DocsLayout {...baseOptions()} tree={pageTree} githubUrl="https://github.com/inngest/inngest">
      <Suspense>{clientLoader.useContent(path)}</Suspense>
    </DocsLayout>
  );
}
