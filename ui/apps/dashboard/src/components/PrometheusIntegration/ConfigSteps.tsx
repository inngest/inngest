import DashboardCodeBlock from '@/components/DashboardCodeBlock/DashboardCodeBlock';
import EnvironmentSelectMenu from '@/components/Navigation/Environments';

// TODO(cdzombak): add "select env only" to env chooser; default to prod
// TODO(cdzombak): build YAML incl token and env name

type Props = {
  metricsGranularitySeconds: number;
};

export default function ConfigSteps({ metricsGranularitySeconds }: Props) {
  const scrapeInterval = Math.max(30, metricsGranularitySeconds / 5).toFixed(0) + 's';
  const scrapeConfigContent = `# add to your Prometheus scrape_configs:
  - job_name: 'inngest-XXX'
    scrape_interval: '${scrapeInterval}'
    honor_labels: true
    static_configs:
      - targets: ['api.inngest.com:443']
    metrics_path: '/v1/prom/XXX'
    scheme: 'https'
    authorization:
      type: 'Bearer'
      credentials: 'signkey-prod-XXX'`;

  return (
    <>
      <div className="text-basis text-lg font-normal">
        <div className="border-subtle ml-3 border-l">
          <div className="before:border-subtle before:text-basis before:bg-canvasBase relative ml-[32px] pb-7 before:absolute before:left-[-46px] before:h-[28px] before:w-[28px] before:rounded-full before:border before:text-center before:align-middle before:text-[13px] before:content-['1']">
            <div className="text-basis mb-4 text-base">
              Select an environment to view its Prometheus{' '}
              <code className="bg-gray-100 p-0.5">scrape_config</code>.
            </div>
            <EnvironmentSelectMenu collapsed={false} />
          </div>
        </div>

        <div className="ml-3">
          <div className="before:border-subtle before:text-basis before:bg-canvasBase relative ml-[32px] pb-5 before:absolute before:left-[-46px] before:h-[28px] before:w-[28px] before:rounded-full before:border before:text-center before:align-middle before:text-[13px] before:content-['2']">
            <div className="text-basis mb-4 text-base">
              Add this item to the <code className="bg-gray-100 p-0.5">scrape_configs</code> section
              of your Prometheus configuration.
            </div>
            <DashboardCodeBlock
              header={{ title: 'scrape_config (environment: XXX)' }}
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
