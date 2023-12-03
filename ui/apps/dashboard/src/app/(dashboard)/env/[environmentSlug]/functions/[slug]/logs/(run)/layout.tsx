'use client';

import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { SlideOver } from '@inngest/components/SlideOver';

type RunLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
    slug: string;
  };
};

export default function RunLayout({ children, params }: RunLayoutProps) {
  const router = useRouter();

  return (
    <SlideOver
      size="large"
      onClose={() =>
        router.push(
          `/env/${params.environmentSlug}/functions/${encodeURIComponent(
            params.slug
          )}/logs` as Route
        )
      }
    >
      {children}
    </SlideOver>
  );
}
