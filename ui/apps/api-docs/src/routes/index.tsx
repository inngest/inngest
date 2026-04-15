import { Suspense } from 'react';
import { createFileRoute, notFound } from '@tanstack/react-router';
import browserCollections from 'collections/browser';
import type { Root } from 'fumadocs-core/page-tree';
import { DocsLayout } from 'fumadocs-ui/layouts/docs';
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from 'fumadocs-ui/page';

import { useMDXComponents } from '@/components/mdx';
import { baseOptions } from '@/lib/layout.shared';
import { source } from '@/lib/source';

type LoaderData = { path: string; pageTree: Root };

export const Route = createFileRoute('/')({
  component: Page,
  loader: async () => {
    const page = source.getPage([]);
    if (!page) throw notFound();

    await clientLoader.preload(page.path);
    return {
      path: page.path,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      pageTree: source.getPageTree() as any,
    };
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
  const { path, pageTree } = Route.useLoaderData() as unknown as LoaderData;

  return (
    <DocsLayout {...baseOptions()} tree={pageTree} githubUrl="https://github.com/inngest/inngest">
      <Suspense>{clientLoader.useContent(path)}</Suspense>
    </DocsLayout>
  );
}
