import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';

export default function AppFAQ() {
  return (
    <AccordionList className="divide-y- rounded-none border-0" type="multiple" defaultValue={[]}>
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
                <p>
                  To integrate your code hosted on another platform with Inngest, you need to inform
                  Inngest about the location of your app and functions.
                </p>
                <p>
                  For example, imagine that your serve() handler is located at /api/inngest, and
                  your domain is myapp.com. In this scenario, you will need to sync your app to
                  inform Inngest that your apps and functions are hosted at
                  https://myapp.com/api/inngest.
                </p>
              </AccordionList.Content>
            </AccordionList.Item>
          </AccordionList>
        </AccordionList.Content>
      </AccordionList.Item>
    </AccordionList>
  );
}
