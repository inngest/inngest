import { NewButton } from '@inngest/components/Button/index';
import { Card } from '@inngest/components/Card/Card';
import { NewLink } from '@inngest/components/Link/Link';
import { Pill } from '@inngest/components/Pill/Pill';

import { type IntegrationPageContent } from './types';

export default function IntegrationPage({
  content,
  publications,
}: {
  content: IntegrationPageContent;
  // TO DO: change this format
  publications: {
    id: string;
    name: string;
    slug: string;
    projects: [];
    enabled: boolean;
  }[];
}) {
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
          <NewLink className="inline-block" size="small" href={content.url}>
            Read documentation
          </NewLink>
        </div>
      </div>
      {publications.map((p, i) => (
        <Card
          key={`${content.title}-publications-${i}`}
          className="my-9"
          accentPosition="left"
          accentColor={p.enabled ? 'bg-primary-intense' : 'bg-disabled'}
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
        <p>Remove {content.title} integration</p>
        <p className="text-subtle mb-6 mt-3 text-sm">
          Permanently remove the {content.title} integration from Inngest
        </p>
        {/* TO DO: Wire button */}
        <NewButton appearance="solid" kind="danger" label={`Remove ${content.title}`} />
      </div>
    </div>
  );
}
