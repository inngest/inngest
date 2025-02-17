import { useState } from 'react';
import { RiInformationLine } from '@remixicon/react';

import DashboardCodeBlock from '@/components/DashboardCodeBlock/DashboardCodeBlock';
import EnvSelectMenu from '@/components/PrometheusIntegration/EnvSelectMenu';
import { type Environment } from '@/utils/environments';

type Props = {
  metricsGranularitySeconds: number;
};

export default function ConfigSteps({ metricsGranularitySeconds }: Props) {
  const [selectedEnv, setSelectedEnv] = useState<Environment | null>(null);
  const scrapeConfigContent = scrapeConfigTmpl(selectedEnv, metricsGranularitySeconds);
  const envName = selectedEnv ? selectedEnv.name : '';

  return (
    <>
      <div className="text-basis text-lg font-normal">
        <div className="border-subtle ml-3 border-l">
          <div className="before:border-subtle before:text-basis before:bg-canvasBase relative ml-[32px] pb-7 before:absolute before:left-[-46px] before:h-[28px] before:w-[28px] before:rounded-full before:border before:text-center before:align-middle before:text-[13px] before:content-['1']">
            <div className="text-basis mb-4 text-base">
              Select an environment to view its Prometheus{' '}
              <code className="bg-gray-100 p-0.5">scrape_config</code>.
            </div>
            <EnvSelectMenu onSelect={setSelectedEnv} />
          </div>
        </div>

        <div className="ml-3">
          <div className="before:border-subtle before:text-basis before:bg-canvasBase relative ml-[32px] pb-5 before:absolute before:left-[-46px] before:h-[28px] before:w-[28px] before:rounded-full before:border before:text-center before:align-middle before:text-[13px] before:content-['2']">
            <div className="text-basis mb-2 text-base">
              Add this item to the <code className="bg-gray-100 p-0.5">scrape_configs</code> section
              of your Prometheus configuration.
            </div>
            <div className="text-muted mb-4 text-base">
              <RiInformationLine className="text-light -mt-1 mr-1 inline-block h-5 w-5" />
              This configuration includes an authentication token, so keep it secure.
            </div>
            <DashboardCodeBlock
              header={{ title: `scrape_config (${envName})` }}
              tab={{
                content: scrapeConfigContent,
                readOnly: true,
                language: 'yaml',
              }}
            />
          </div>
        </div>
      </div>
    </>
  );
}

function scrapeConfigTmpl(env: Environment | null, metricsGranularitySeconds: number) {
  if (!env) {
    return '# add to your Prometheus scrape_configs:';
  }

  const scrapeInterval = Math.max(30, metricsGranularitySeconds / 5).toFixed(0) + 's';

  return `# add to your Prometheus scrape_configs:
  - job_name: 'inngest-${env.slug}'
    scrape_interval: '${scrapeInterval}'
    honor_labels: true
    honor_timestamps: true
    static_configs:
      - targets: ['api.inngest.com:443']
    metrics_path: '/v1/prom/${env.slug}'
    scheme: 'https'
    authorization:
      type: 'Bearer'
      credentials: '${env.webhookSigningKey}'`;
}
