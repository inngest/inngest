import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { InlineCode } from '@inngest/components/InlineCode';

export default function AppFAQ() {
  return (
    <AccordionList className="rounded-none border-0" type="multiple" defaultValue={[]}>
      <AccordionList.Item value="FAQ">
        <AccordionList.Trigger className="text-muted text-sm data-[state=open]:border-0">
          NEED HELP SETTING UP YOUR APP?
        </AccordionList.Trigger>
        <AccordionList.Content className="px-0">
          <AccordionList type="multiple" defaultValue={[]}>
            <AccordionList.Item value="app_definition">
              <AccordionList.Trigger>What is an app?</AccordionList.Trigger>
              <AccordionList.Content>
                <p>Inngest “App” is a group of functions served on a single endpoint or server. </p>
              </AccordionList.Content>
            </AccordionList.Item>
            <AccordionList.Item value="syncing_app">
              <AccordionList.Trigger>What does “syncing an app” mean?</AccordionList.Trigger>
              <AccordionList.Content>
                <p className="mb-2">
                  Your Inngest functions are defined and execute within your application. To enable
                  Inngest to fetch your function configuration and invoke functions, it must be able
                  to reach your app "Syncing" tells Inngest where your app is running.
                </p>
                <p className="mb-2">
                  Syncing an app works by providing Inngest with the URL of your application's{' '}
                  <InlineCode value="serve()" /> handler endpoint, typically at{' '}
                  <InlineCode value="&lt;your-hostname&gt;/api/inngest" />. When you sync an app,
                  Inngest reads the configuration of your app and functions and stores the URL to
                  send future invocation requests.
                </p>
                <p>
                  As your functions may change, it is necessary to sync your app whenever it
                  changes. The Inngest Dev Server does this by polling for changes every 5 seconds
                  by default.
                </p>
              </AccordionList.Content>
            </AccordionList.Item>
          </AccordionList>
        </AccordionList.Content>
      </AccordionList.Item>
    </AccordionList>
  );
}
