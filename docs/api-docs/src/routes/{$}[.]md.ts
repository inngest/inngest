import { createFileRoute } from '@tanstack/react-router';

import { getLLMText } from '@/lib/get-llm-text';
import { markdownPathToSlugs, source } from '@/lib/source';

export const Route = createFileRoute('/{$}.md')({
  server: {
    handlers: {
      GET: async ({ params }) => {
        const splat = (params as { _splat?: string })._splat ?? '';
        const slugs = markdownPathToSlugs(splat.split('/').filter(Boolean));
        const page = source.getPage(slugs);
        if (!page) {
          return new Response('Not Found', {
            status: 404,
            headers: { 'Content-Type': 'text/plain; charset=utf-8' },
          });
        }

        return new Response(await getLLMText(page), {
          headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
        });
      },
    },
  },
});
