import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';
import { Link } from '@inngest/components/Link';
import { Pill } from '@inngest/components/Pill';
import { useNavigate } from '@tanstack/react-router';
import { toast } from 'sonner';

import { type IntegrationPageContent, type Publication } from './types';

const statusLabel = (pub: Publication) => {
  if (pub.enabled) {
    return 'Active';
  }
  if (pub.status === 'ERROR') {
    return 'Error';
  }
  if (pub.status === 'SETUP_INCOMPLETE') {
    return 'Setup Incomplete';
  }
  if (pub.status === 'STOPPED') {
    return 'Stopped';
  }
  return 'Disabled';
};

const statusKind = (pub: Publication): 'primary' | 'default' | 'error' => {
  if (pub.enabled) {
    return 'primary';
  }
  if (pub.status === 'ERROR' || pub.status === 'SETUP_INCOMPLETE') {
    return 'error';
  }
  return 'default';
};

export default function IntegrationPage({
  content,
  publications,
  onDelete,
}: {
  content: IntegrationPageContent;
  onDelete: (id: string) => Promise<{ success: boolean; error: string | null }>;
  publications: Publication[];
}) {
  const navigate = useNavigate();
  const [isDeleting, setIsDeleting] = useState(false);

  const successRedirect = () => {
    navigate({ to: '/settings/integrations' });
  };

  const brokenIntegrations = publications.some(
    (p) => p.status === 'ERROR' || p.status === 'SETUP_INCOMPLETE'
  );

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
      {brokenIntegrations && (
        <div className="border-muted pt-6">
          <p className="text-tertiary-moderate text-sm">
            The integration setup did not complete successfully. Remove and re-add the integration
            to try again.
          </p>
        </div>
      )}
      {publications.map((p, i) => {
        const status = statusKind(p);
        return (
          <Card
            key={`${content.title}-publications-${i}`}
            className="my-9"
            accentPosition="left"
            accentColor={
              status == 'error'
                ? 'bg-tertiary-moderate'
                : p.enabled
                ? 'bg-primary-intense'
                : 'bg-surfaceMuted'
            }
          >
            <Card.Content className="p-6">
              <div className="flex flex-row items-center justify-between">
                <div className="flex flex-col">
                  <div>
                    <Pill appearance="solid" kind={statusKind(p)}>
                      {statusLabel(p)}
                    </Pill>
                  </div>
                  <div className="mt-4 flex flex-row items-center justify-start">
                    <div className="text-basis text-lg font-medium">{p.name}</div>
                  </div>
                </div>
              </div>
            </Card.Content>
          </Card>
        );
      })}

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
          loading={isDeleting}
          onClick={async () => {
            if (!publications || publications.length === 0) {
              console.error('no cdc connection to remove');
              return;
            }

            setIsDeleting(true);
            let lastError: string | null = null;

            for (const pub of publications) {
              const { error } = await onDelete(pub.id);
              if (error) {
                lastError = error;
              }
            }

            setIsDeleting(false);

            if (lastError) {
              toast.error(lastError);
            } else {
              successRedirect();
            }
          }}
        />
      </div>
    </div>
  );
}
