'use client';

import { Card } from '@inngest/components/Card/Card';
import { Link } from '@inngest/components/Link/Link';
import { IconPrometheus } from '@inngest/components/icons/platforms/Prometheus';

import DashboardCodeBlock from '@/components/DashboardCodeBlock/DashboardCodeBlock';
import EnvironmentSelectMenu from '@/components/Navigation/Environments';

// TODO(cdzombak): display current limits
// TODO(cdzombak): free/non-entitled plan handling
// TODO(cdzombak): use current limit / 4 for scrape interval
// TODO(cdzombak): Link to docs

export default function PrometheusSetupPage() {
  // TODO(cdzombak): add "select env only" to env chooser; default to prod
  // TODO(cdzombak): build YAML incl token

  const scrapeConfigContent = `# add to your Prometheus scrape_configs:
  - job_name: 'inngest-ENVSLUG'
    scrape_interval: '30s'
    honor_labels: true
    static_configs:
      - targets: ['api.inngest.com:443']
    metrics_path: '/v1/prom/ENVSLUG'
    scheme: 'https'
    authorization:
      type: 'Bearer'
      credentials: 'signkey-prod-XXX'`;

  return (
    <div className="mx-auto mt-16 flex w-[800px] flex-col">
      <div className="text-basis mb-7 flex flex-row items-center justify-start text-2xl font-medium">
        <div className="bg-contrast mr-4 flex h-12 w-12 items-center justify-center rounded">
          <IconPrometheus className="text-onContrast" size={20} />
        </div>
        Prometheus
      </div>
      <div className="text-muted w-full text-base font-normal">
        This integration allows your Prometheus server to scrape metrics about your Inngest
        deployment.
        {/*<Link target="_blank" size="medium" href="https://www.inngest.com/docs/deploy/vercel">*/}
        {/*  Read documentation*/}
        {/*</Link>*/}
      </div>

      <Card className="my-6" accentPosition="left" accentColor={'bg-errorContrast'}>
        <Card.Content className="p-6">
          Your Inngest plan does not support Prometheus integration. To enable this feature,{' '}
          <Link size="medium" href="/billing" className="inline">
            upgrade your plan
          </Link>
          .
        </Card.Content>
      </Card>

      <div className="text-basis text-lg font-normal">
        <div className="border-subtle ml-3 border-l">
          <div className="before:border-subtle before:text-basis before:bg-canvasBase relative ml-[32px] pb-7 before:absolute before:left-[-46px] before:h-[28px] before:w-[28px] before:rounded-full before:border before:text-center before:align-middle before:text-[13px] before:content-['1']">
            <div className="text-basis mb-4 text-base">
              Select an environment to view its Prometheus <code>scrape_config</code>.
            </div>
            <EnvironmentSelectMenu collapsed={false} />
          </div>
        </div>

        <div className="ml-3">
          <div className="before:border-subtle before:text-basis before:bg-canvasBase relative ml-[32px] pb-5 before:absolute before:left-[-46px] before:h-[28px] before:w-[28px] before:rounded-full before:border before:text-center before:align-middle before:text-[13px] before:content-['2']">
            <div className="text-basis text-base ">
              Add this item to the <code>scrape_configs</code> section of your Prometheus
              configuration.
            </div>
          </div>
        </div>

        <DashboardCodeBlock
          header={{ title: 'scrape_config (environment: ENVSLUG)' }}
          tab={{
            content: scrapeConfigContent,
            readOnly: true,
            language: 'yaml',
          }}
        />
      </div>
    </div>
  );
}
