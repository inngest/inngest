import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { Link } from '@inngest/components/Link';

export default function CommonVercelErrors() {
  return (
    <div className="my-6">
      <p className="bg-canvasSubtle border-subtle text-subtle rounded-t-lg border border-b-0 px-3 py-2 text-sm font-medium">
        {' '}
        Why might the syncs fail, and how can I resolve it?
      </p>
      <AccordionList type="multiple" defaultValue={[]} className="rounded-t-none">
        <AccordionList.Item value="protection-key" className="first:rounded-t-none">
          <AccordionList.Trigger className="text-subtle text-sm">
            Deployment protection key is enabled
          </AccordionList.Trigger>

          <AccordionList.Content className="text-subtle">
            <p>
              Inngest may not be able to communicate with your application by default. The sync can
              fail if the deployment protection key isn&apos;t bypassed.{' '}
              <Link
                className="inline"
                size="small"
                target="_blank"
                href="https://www.inngest.com/docs/deploy/vercel#bypassing-deployment-protection?ref=app-onboarding-sync-app"
              >
                Learn how to bypass it
              </Link>
            </p>
          </AccordionList.Content>
        </AccordionList.Item>
        <AccordionList.Item value="prod-app">
          <AccordionList.Trigger className="text-subtle text-sm">
            Your Inngest app isn&apos;t merged to production
          </AccordionList.Trigger>

          <AccordionList.Content className="text-subtle">
            <p>
              Only Vercel production deploys will show up in your Inngest production environment. If
              your Inngest app is only set up on a Vercel preview, it will appear as an Inngest
              branch preview. You can open a branch environment using the environment dropdown at
              the top left of the dashboard.
            </p>
          </AccordionList.Content>
        </AccordionList.Item>
      </AccordionList>
    </div>
  );
}
