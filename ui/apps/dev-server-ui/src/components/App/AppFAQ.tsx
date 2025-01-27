import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { InlineCode } from '@inngest/components/Code';
import { CodeLine } from '@inngest/components/CodeLine';
import { Link } from '@inngest/components/Link';

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
              <AccordionList.Trigger>What does “syncing" an app mean?</AccordionList.Trigger>
              <AccordionList.Content>
                <p className="mb-2">
                  As your Inngest functions are defined and execute within your application, it is
                  necessary for Inngest to be able communicate with your application to 1) read your
                  functions' configurations and 2) invoke functions.
                </p>
                <p className="mb-2">
                  "<strong>Syncing</strong>" an app establishes a connection via HTTP at the correct
                  URL endpoint and synchronizes configuration to Inngest.
                </p>
                <p className="mb-2">
                  Syncing an app works by providing Inngest with the URL of your application's{' '}
                  <InlineCode>serve()</InlineCode> handler endpoint, typically at{' '}
                  <InlineCode>&lt;your-hostname&gt;/api/inngest</InlineCode>. When you sync an app,
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
            <AccordionList.Item value="polling">
              <AccordionList.Trigger>
                Why is my app being polled every few seconds?
              </AccordionList.Trigger>
              <AccordionList.Content>
                <p className="mb-2">
                  The Dev Server polls your app's serve endpoint every few seconds to check for new
                  functions or updates to function configurations. This enables a "hot reload" like
                  experience for your Inngest functions.
                </p>
                <p className="mb-2">
                  You can adjust the polling interval using{' '}
                  <InlineCode>--poll-interval &lt;seconds&gt;</InlineCode> or disable it completely
                  with the <InlineCode>--no-poll</InlineCode> flag.
                </p>
              </AccordionList.Content>
            </AccordionList.Item>
            <AccordionList.Item value="auto_discovery">
              <AccordionList.Trigger>
                Why are other URLs not in my app being polled?
              </AccordionList.Trigger>
              <AccordionList.Content>
                <p className="mb-2">
                  The Dev Server will automatically discover and sync apps running on common ports
                  and paths. This includes ports like 3000, 5000, 8080, and endpoints like{' '}
                  <InlineCode>/api/inngest</InlineCode> and <InlineCode>/x/inngest</InlineCode>.{' '}
                  <Link
                    target="_blank"
                    size="small"
                    className="inline"
                    href="https://www.inngest.com/docs/dev-server#auto-discovery"
                  >
                    Learn more in the docs
                  </Link>
                  .
                </p>
                <p className="mb-2">
                  You can disable auto-discovery with the <InlineCode>--no-discovery</InlineCode>{' '}
                  flag.
                </p>
              </AccordionList.Content>
            </AccordionList.Item>
            <AccordionList.Item value="skip_manual_sync">
              <AccordionList.Trigger>How can I skip manual syncing?</AccordionList.Trigger>
              <AccordionList.Content>
                <p className="mb-2">
                  You can specify the URL of your apps at startup by using the{' '}
                  <InlineCode>-u &lt;url&gt;</InlineCode> flag. You can specify more than one app
                  URLs by using the flag multiple times. For example:
                </p>
                <CodeLine
                  code="inngest dev -u http://localhost:3000/api/inngest -u http://localhost:3333/api/inngest"
                  className="mb-2"
                />
                <p className="mb-2">
                  Alternatively, you can specify the URL of your app in an{' '}
                  <InlineCode>inngest.json</InlineCode> configuration file that you can check into
                  version control.{' '}
                  <Link
                    target="_blank"
                    size="small"
                    className="inline"
                    href="https://www.inngest.com/docs/dev-server#configuration-file"
                  >
                    Learn more in the docs
                  </Link>
                  .
                </p>
              </AccordionList.Content>
            </AccordionList.Item>
          </AccordionList>
        </AccordionList.Content>
      </AccordionList.Item>
    </AccordionList>
  );
}
