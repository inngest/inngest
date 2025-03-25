'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button/index';
import { Card } from '@inngest/components/Card/Card';
import { Link } from '@inngest/components/Link/Link';
import { Pill } from '@inngest/components/Pill/Pill';
import { toast } from 'sonner';

import { type IntegrationPageContent, type Publication } from './types';

export default function IntegrationPage({
  content,
  publications,
  onDelete,
}: {
  content: IntegrationPageContent;
  onDelete: (id: string) => Promise<{ success: boolean; error: string | null }>;
  publications: Publication[];
}) {
  const router = useRouter();

  const successRedirect = () => {
    router.push('/settings/integrations');
  };

  return (
    <div className="mx-auto mt-6 flex w-[800px] flex-col p-8">
      <div className="flex flex-col">
        <div className="mb-5 flex items-center">
          <div className="bg-contrast mr-4 flex h-[52px] w-[52px] items-center justify-center rounded">
            {content.logo}
          </div>
          <div className="text-basis text-xl font-medium">{content.title}</div>
        </div>

        <div className="text-subtle text-sm">
          Manage your {content.title} integration from this page.{' '}
          <Link className="inline-block" size="small" href={content.url} target="_blank">
            Read documentation
          </Link>
        </div>
      </div>
      {publications.map((p, i) => (
        <Card
          key={`${content.title}-publications-${i}`}
          className="my-9"
          accentPosition="left"
          accentColor={p.enabled ? 'bg-primary-intense' : 'bg-surfaceMuted'}
        >
          <Card.Content className="p-6">
            <div className="flex flex-row items-center justify-between">
              <div className="flex flex-col">
                <div>
                  <Pill appearance="solid" kind={p.enabled ? 'primary' : 'default'}>
                    {p.enabled ? 'Active' : 'Disabled'}
                  </Pill>
                </div>
                <div className="mt-4 flex flex-row items-center justify-start">
                  <div className="text-basis text-lg font-medium">{p.name}</div>
                </div>
              </div>
            </div>
          </Card.Content>
        </Card>
      ))}

      <div className="border-muted border-t py-7">
        <div className="flex items-center gap-2">
          <p>Remove {content.title} integration</p>
        </div>
        <p className="text-subtle mb-6 mt-3 text-sm">
          Permanently remove the {content.title} integration from Inngest
        </p>
        <Button
          appearance="solid"
          kind="danger"
          label={`Remove ${content.title}`}
          onClick={async () => {
            if (!publications || !publications[0]) {
              console.error('no neon cdc connection to remove');
              return;
            }

            const { success, error } = await onDelete(publications[0].id);
            if (success) {
              successRedirect();
            }
            if (error) {
              toast.error(error);
            }
          }}
        />
      </div>
    </div>
  );
}
