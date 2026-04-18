import { Suspense } from 'react';
import { createFileRoute, notFound } from '@tanstack/react-router';
import browserCollections from 'collections/browser';
import type { Root } from 'fumadocs-core/page-tree';
import { DocsLayout } from 'fumadocs-ui/layouts/docs';
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from 'fumadocs-ui/page';

import { useMDXComponents } from '@/components/mdx';
import { baseOptions } from '@/lib/layout.shared';
import { getDocPage } from '@/lib/page-data';

type LoaderData = { path: string; pageTree: Root };

export const Route = createFileRoute('/')({
  component: Page,
  loader: async () => {
    const data = await getDocPage({ data: { slugs: [] } });
    if (!data) throw notFound();

    await clientLoader.preload(data.path);
    return data as LoaderData;
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
