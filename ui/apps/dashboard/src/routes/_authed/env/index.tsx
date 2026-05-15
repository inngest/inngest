import Environments from '@/components/Environments/Environments';
import { Header } from '@inngest/components/Header/Header';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/_authed/env/')({
  component: EnvComponent,
});

function EnvComponent() {
  return (
    <div className="flex-col">
      <Header backNav={true} breadcrumb={[{ text: 'All Environments' }]} />
      <div className="no-scrollbar overflow-y-scroll px-6">
        <Environments />
      </div>
    </div>
  );
}
