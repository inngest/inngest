import { Suspense } from 'react';
import { createFileRoute, notFound } from '@tanstack/react-router';
import { createServerFn } from '@tanstack/react-start';
import browserCollections from 'collections/browser';
import { useFumadocsLoader } from 'fumadocs-core/source/client';
import { DocsLayout } from 'fumadocs-ui/layouts/docs';
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from 'fumadocs-ui/page';

import { useMDXComponents } from '@/components/mdx';
import { baseOptions } from '@/lib/layout.shared';
import { source } from '@/lib/source';

export const Route = createFileRoute('/docs/$')({
  component: Page,
  loader: async ({ params }) => {
    const slugs = (params as { _splat?: string })._splat?.split('/').filter(Boolean) ?? [];
    const data = await serverLoader({ data: slugs });
    await clientLoader.preload(data.path);
    return data;
  },
});

const serverLoader = createServerFn({
  method: 'GET',
})
  .inputValidator((slugs: string[]) => slugs)
  .handler(async ({ data: slugs }) => {
    const page = source.getPage(slugs);
    if (!page) throw notFound();

    return {
      path: page.path,
      pageTree: await source.serializePageTree(source.getPageTree()),
    };
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
  const data = useFumadocsLoader(Route.useLoaderData());

  return (
    <DocsLayout
      {...baseOptions()}
      tree={data.pageTree}
      githubUrl="https://github.com/inngest/inngest"
    >
      <Suspense>{clientLoader.useContent(data.path)}</Suspense>
    </DocsLayout>
  );
}
