import { useEffect, useState } from 'react';
import { createClientAPIPage, type ClientApiPagePayload } from 'fumadocs-openapi/ui/create-client';
import * as TabsComponents from 'fumadocs-ui/components/tabs';
import defaultMdxComponents from 'fumadocs-ui/mdx';
import type { MDXComponents } from 'mdx/types';

const ClientAPIPage = createClientAPIPage();

// Simple cache so we don't refetch the same spec on every render
const specCache = new Map<string, ClientApiPagePayload['bundled']>();

async function fetchSpec(url: string): Promise<ClientApiPagePayload['bundled']> {
  const cached = specCache.get(url);
  if (cached) return cached;
  const res = await fetch(url);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const data = (await res.json()) as ClientApiPagePayload['bundled'];
  specCache.set(url, data);
  return data;
}

// Props that match the MDX output from fumadocs-openapi's generateFiles()
type APIPageProps = {
  document: string;
  operations?: { path: string; method: string }[];
  webhooks?: { name: string; method: string }[];
  showTitle?: boolean;
  showDescription?: boolean;
};

// Bridges the MDX-generated `document` URL prop to the `payload.bundled` object
// that ClientAPIPage expects. Fetches the spec at runtime from the public URL.
function APIPage({ document: documentUrl, ...rest }: APIPageProps) {
  const [payload, setPayload] = useState<ClientApiPagePayload | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetchSpec(documentUrl)
      .then((bundled) => {
        if (!cancelled) setPayload({ bundled });
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(String(err));
      });
    return () => {
      cancelled = true;
    };
  }, [documentUrl]);

  if (error) {
    return (
      <div className="rounded border border-red-200 bg-red-50 p-4 text-sm text-red-700">
        Failed to load API spec: {error}
      </div>
    );
  }
  if (!payload) {
    return <div className="text-fd-muted-foreground p-4 text-sm">Loading API reference…</div>;
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return <ClientAPIPage payload={payload} {...(rest as any)} />;
}

export function getMDXComponents(components?: MDXComponents): MDXComponents {
  return {
    ...defaultMdxComponents,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    APIPage: APIPage as any,
    ...TabsComponents,
    ...components,
  } satisfies MDXComponents;
}

export const useMDXComponents = getMDXComponents;

declare global {
  type MDXProvidedComponents = ReturnType<typeof getMDXComponents>;
}
