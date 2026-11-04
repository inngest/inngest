import { Button } from '@inngest/components/Button';
import { QuietCopyButton } from './QuietCopyButton';
import { Link } from '@inngest/components/Link';
import { RiCheckboxCircleFill, RiInformationLine } from '@remixicon/react';
import { useState } from 'react';
import CommandBlock from '@inngest/components/CodeBlock/CommandBlock';
import { Select } from '@inngest/components/Select/Select';
import AddAppButton from '@/components/App/AddAppButton';
import { useQuickstart } from './QuickstartProvider';
import { useQuickstartTracking } from './useQuickstartTracking';

export function ManualSetup({
  discoveryEnabled,
  disconnected,
  connectedAppNames = [],
}: {
  discoveryEnabled: boolean;
  disconnected: boolean;
  connectedAppNames?: string[];
}) {
  const { server } = useQuickstart();
  const connected = connectedAppNames.length > 0;
  const connectedMessage =
    connectedAppNames.length === 1
      ? `${connectedAppNames[0]} is connected.`
      : `${connectedAppNames[0]} and ${connectedAppNames.length - 1} more ${
          connectedAppNames.length === 2 ? 'app' : 'apps'
        } are connected.`;
  const track = useQuickstartTracking(server);
  const [language, setLanguage] = useState('TypeScript');
  const [manager, setManager] = useState('npm');
  const managers =
    language === 'TypeScript'
      ? ['npm', 'yarn', 'pnpm', 'bun']
      : language === 'Python'
      ? ['pip', 'uv']
      : ['go'];
  const packageManager = managers.includes(manager) ? manager : managers[0];
  const commands: Record<string, string> = {
    npm: 'npm install inngest',
    yarn: 'yarn add inngest',
    pnpm: 'pnpm add inngest',
    bun: 'bun add inngest',
    pip: 'pip install inngest',
    uv: 'uv add inngest',
    go: 'go get github.com/inngest/inngestgo',
  };
  const tabs = managers.map((title) => ({
    title,
    content: commands[title],
    language: 'shell',
  }));
  const snippet = tabs.find((tab) => tab.title === packageManager) || tabs[0];
  const guide =
    language === 'TypeScript' ? 'typescript' : language.toLowerCase();
  return (
    <div className="space-y-8">
      <section className="space-y-3" aria-labelledby="install-sdk">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1">
            <h2 id="install-sdk" className="font-medium">
              1. Install the SDK
            </h2>
            <p className="text-muted text-sm">
              Install Inngest in your project directory.
            </p>
          </div>
          <Select
            label="Language"
            size="small"
            value={{ id: language, name: language }}
            onChange={(option) => setLanguage(option.id)}
          >
            <Select.Button size="small">{language}</Select.Button>
            <Select.Options>
              {['TypeScript', 'Python', 'Go'].map((name) => (
                <Select.Option key={name} option={{ id: name, name }}>
                  {name}
                </Select.Option>
              ))}
            </Select.Options>
          </Select>
        </div>
        <CommandBlock.Wrapper>
          <CommandBlock.Header className="flex flex-wrap items-center gap-y-1 pb-1 pr-4 sm:pb-0">
            <div className="max-w-full overflow-x-auto">
              <CommandBlock.Tabs
                tabs={tabs}
                activeTab={packageManager}
                setActiveTab={setManager}
              />
            </div>
            <div className="ml-auto">
              <QuietCopyButton
                onCopy={() =>
                  track('manual_install_copy_clicked', {
                    language,
                    package_manager: packageManager,
                  })
                }
                content={snippet?.content}
              />
            </div>
          </CommandBlock.Header>
          <CommandBlock currentTabContent={snippet} />
        </CommandBlock.Wrapper>
      </section>
      <section className="space-y-3" aria-labelledby="connect-app">
        <div className="space-y-1">
          <h2 id="connect-app" className="font-medium">
            2. Connect your app
          </h2>
          <p className="text-muted text-sm">
            Create a function and start your application.{' '}
            <Link
              className="inline-flex max-w-full align-baseline"
              onClick={() =>
                track('docs_clicked', { guide: 'manual', language })
              }
              href={`https://www.inngest.com/docs/learn/serving-inngest-functions?guide=${guide}&ref=dev-server`}
            >
              Open {language} setup guide
            </Link>
          </p>
        </div>
        <div className="border-subtle bg-canvasSubtle flex flex-wrap items-center justify-between gap-3 rounded border p-3">
          <p
            className="text-muted flex min-w-0 flex-[1_1_180px] items-start gap-2 text-xs"
            role="status"
          >
            {connected ? (
              <RiCheckboxCircleFill
                aria-hidden="true"
                className="text-success mt-0.5 h-3.5 w-3.5 shrink-0"
              />
            ) : (
              <RiInformationLine
                aria-hidden="true"
                className="mt-0.5 h-3.5 w-3.5 shrink-0"
              />
            )}
            <span className="break-words [overflow-wrap:anywhere]">
              {connected
                ? connectedMessage
                : disconnected
                ? 'Your app is disconnected. Start it again or sync its URL.'
                : discoveryEnabled
                ? 'Auto-detecting apps…'
                : 'Automatic discovery is disabled. Sync your app URL manually.'}
            </span>
          </p>
          {connected ? (
            <Button
              href="/apps"
              label="View apps"
              kind="secondary"
              appearance="outlined"
            />
          ) : (
            <AddAppButton secondary label="Sync manually" />
          )}
        </div>
        {disconnected && (
          <Link className="inline-flex" href="/apps" target="_self">
            View app connection errors
          </Link>
        )}
      </section>
    </div>
  );
}
