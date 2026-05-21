import { useEffect, useState, type ReactNode } from 'react';
import { createClientAPIPage, type ClientApiPagePayload } from 'fumadocs-openapi/ui/create-client';
import { Callout, type CalloutType } from 'fumadocs-ui/components/callout';
import * as TabsComponents from 'fumadocs-ui/components/tabs';
import defaultMdxComponents from 'fumadocs-ui/mdx';
import type { MDXComponents } from 'mdx/types';

const ClientAPIPage = createClientAPIPage();

// Soft-tag extensions (set by scripts/generate-docs.ts) rendered as Callouts at
// the top of an operation page. Add new entries to introduce additional badges.
const OPERATION_CALLOUTS: { key: string; type: CalloutType; title: string; body: ReactNode }[] = [
  {
    key: 'x-beta',
    type: 'warn',
    title: 'Beta',
    body: 'This endpoint is in Beta. Behavior and shape may change before general availability.',
  },
];

function OperationCallouts({
  bundled,
  operations,
}: {
  bundled: ClientApiPagePayload['bundled'];
  operations?: { path: string; method: string }[];
}) {
  if (!operations?.length) return null;
  const paths = bundled?.paths as
    | Record<string, Record<string, Record<string, unknown>>>
    | undefined;
  if (!paths) return null;
  const active = OPERATION_CALLOUTS.filter(({ key }) =>
    operations.some((op) => paths[op.path]?.[op.method.toLowerCase()]?.[key])
  );
  if (active.length === 0) return null;
  return (
    <>
      {active.map(({ key, type, title, body }) => (
        <Callout key={key} type={type} title={title}>
          {body}
        </Callout>
      ))}
    </>
  );
}

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

  return (
    <>
      <OperationCallouts bundled={payload.bundled} operations={rest.operations} />
      {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
      <ClientAPIPage payload={payload} {...(rest as any)} />
    </>
  );
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
