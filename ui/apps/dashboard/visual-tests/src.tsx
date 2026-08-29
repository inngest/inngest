import { useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import colors from 'tailwindcss/colors';

import SimpleLineChart from '@/components/Charts/SimpleLineChart';
import { AccountConcurrency } from '@/components/Metrics/AccountConcurrency';
import { Backlog } from '@/components/Metrics/Backlog';
import { Concurrency } from '@/components/Metrics/Concurrency';
import { sumScopedMetricsByGroup } from '@/components/Metrics/metricAggregation';
import { RunsThrougput } from '@/components/Metrics/RunsThroughput';
import '@inngest/components/AppRoot/globals.css';
import { TooltipProvider } from '@inngest/components/Tooltip';

import {
  accountConcurrency,
  entities,
  functionBacklogData,
  functionGaugeData,
  workspaceFixture,
} from './fixtures';
import './visual-tests.css';

const appConcurrency = sumScopedMetricsByGroup(
  workspaceFixture.allFunctionConcurrency.metrics,
  ({ id }) => entities[id as keyof typeof entities]?.appID,
);

function VisualCase({
  id,
  children,
}: {
  id: string;
  children: React.ReactNode;
}) {
  return (
    <section className="visual-case" data-testid={id}>
      {children}
    </section>
  );
}

function App() {
  useEffect(() => {
    let frame = 0;
    const markReady = () => {
      frame += 1;
      if (frame === 3) {
        document.documentElement.dataset.visualReady = 'true';
        return;
      }
      requestAnimationFrame(markReady);
    };
    requestAnimationFrame(markReady);
  }, []);

  return (
    <main>
      <VisualCase id="function-running-gaps">
        <SimpleLineChart
          animation={false}
          connectNulls
          data={functionGaugeData}
          desc="Gauge observations and concurrency limit periods."
          height={240}
          isLoading={false}
          legend={[
            {
              name: 'Concurrency Limit Hit',
              dataKey: 'concurrencyLimit',
              color: colors.amber[500],
              referenceArea: true,
            },
            {
              name: 'Running',
              dataKey: 'running',
              color: colors.blue[500],
            },
          ]}
          title="Step Running - Point in Time"
        />
      </VisualCase>

      <VisualCase id="function-backlog-independent-gaps">
        <SimpleLineChart
          animation={false}
          connectNulls
          data={functionBacklogData}
          height={240}
          isLoading={false}
          legend={[
            {
              name: 'Queued',
              dataKey: 'scheduled',
              color: colors.sky[500],
            },
            {
              name: 'Sleeping',
              dataKey: 'sleeping',
              color: colors.teal[500],
            },
          ]}
          title="Function Backlog"
        />
      </VisualCase>

      <VisualCase id="scoped-throughput-independent-gaps">
        <RunsThrougput
          animation={false}
          entities={entities}
          workspace={workspaceFixture}
        />
      </VisualCase>

      <VisualCase id="app-backlog-leading-trailing-gaps">
        <Backlog
          animation={false}
          entities={entities}
          workspace={workspaceFixture}
        />
      </VisualCase>

      <VisualCase id="app-concurrency-aggregated-zeroes">
        <Concurrency
          animation={false}
          entities={entities}
          isMarketplace
          metrics={appConcurrency}
        />
      </VisualCase>

      <VisualCase id="account-concurrency-limit">
        <AccountConcurrency
          accountConcurrency={accountConcurrency}
          animation={false}
          isMarketplace
          limit={50}
        />
      </VisualCase>
    </main>
  );
}

createRoot(document.getElementById('root')!).render(
  <TooltipProvider>
    <App />
  </TooltipProvider>,
);
