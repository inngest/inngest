import { createFileRoute } from '@tanstack/react-router';

import { InfraDashboard } from '@/components/InfraDashboard/InfraDashboard';

export const Route = createFileRoute('/_authed/env/$envSlug/')({
  component: EnvHome,
});

function EnvHome() {
  return <InfraDashboard />;
}
