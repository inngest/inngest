import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { NewLink } from '@inngest/components/Link';

export default function CommonVercelErrors() {
  return (
    <div className="my-6">
      <p className="mb-3 text-sm font-medium">
        {' '}
        Why might the syncs fail, and how can I resolve it?
      </p>
      <AccordionList type="multiple" defaultValue={[]}>
        <AccordionList.Item value="protection-key">
          <AccordionList.Trigger className="text-subtle text-sm">
            Deployment protection key is enabled
          </AccordionList.Trigger>

          <AccordionList.Content className="text-subtle">
            <p>
              Inngest may not be able to communicate with your application by default. The sync can
              fail if the deployment protection key isn&apos;t bypassed.{' '}
              <NewLink
                className="inline"
                size="small"
                target="_blank"
                href="https://www.inngest.com/docs/deploy/vercel#bypassing-deployment-protection?ref=app-onboarding-sync-app"
              >
                Learn how to bypass it
              </NewLink>
            </p>
          </AccordionList.Content>
        </AccordionList.Item>
        <AccordionList.Item value="feature-branch">
          <AccordionList.Trigger className="text-subtle text-sm">
            Your app is on a feature branch
          </AccordionList.Trigger>

          <AccordionList.Content className="text-subtle">
            <p>
              Syncing may not happen if your app is on a feature branch. To fix this, use the manual
              sync option to sync your app.
            </p>
          </AccordionList.Content>
        </AccordionList.Item>
      </AccordionList>
    </div>
  );
}
