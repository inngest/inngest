import type { UrlObject } from 'url';
import NextLink from 'next/link';
import { Card } from '@inngest/components/Card';

type HelperCardProps = {
  title: string;
  description: string;
  icon: React.ReactNode;
  href: string | UrlObject;
};

export default function HelperCard({ title, description, icon, href }: HelperCardProps) {
  return (
    <Card className="hover:bg-canvasSubtle">
      <NextLink className="block p-4" href={href}>
        {icon}
        <p className="mb-1 mt-3">{title}</p>
        <p className="text-muted text-sm">{description}</p>
      </NextLink>
    </Card>
  );
}
