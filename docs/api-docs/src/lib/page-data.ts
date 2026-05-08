import { createServerFn } from '@tanstack/react-start';

/**
 * Server function that loads page metadata from the fumadocs source.
 *
 * Keeping this in a server function ensures that `@/lib/source` (which
 * imports the eagerly-loaded server collection with all MDX files) never
 * ends up in the client bundle. Without this, client-side navigation would
 * trigger the server collection and load every MDX file into the browser.
 */
export const getDocPage = createServerFn()
  .inputValidator((data: { slugs: string[] }) => data)
  .handler(async ({ data }) => {
    const { source } = await import('./source');
    const page = source.getPage(data.slugs);
    if (!page) return null;
    return {
      path: page.path,
      // getPageTree() returns ReactNode names per the TypeScript type, but
      // for this content (API docs from OpenAPI) all names are plain strings
      // and serialize cleanly via TanStack Start's SuperJSON transport.
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      pageTree: source.getPageTree() as any,
    };
  });
