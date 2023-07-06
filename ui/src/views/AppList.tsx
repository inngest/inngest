import AppCard from '@/components/App/AppCard';
import { IconFunction, IconSpinner } from '@/icons';
import AddAppButton from '@/components/App/AddAppButton';
import { useGetAppsQuery } from '@/store/generated';

const cenas = [{
  autodiscovered: false,
  connected: false,
  functionCount: 0,
  id: '1cena',
  name: 'fake cenas',
  sdkLanguage: 'a',
  sdkVersion: '2.12.0',
  url: 'http://localhost:2020'
  }]


export default function AppList() {
  const { data } = useGetAppsQuery();
  const apps = cenas;

// export default function AppList() {
//   const { data } = useGetAppsQuery();
//   const apps = data?.apps || [];

  return (
    <div className="px-10 py-6 h-full flex flex-col overflow-y-scroll">
      <header className="mb-8">
        <h1 className="text-lg text-slate-50">Connected Apps</h1>
        <p className="my-4">
          This is a list of all apps. We auto-detect apps that you have defined
          in specific ports.
        </p>
        <div className="flex items-center gap-5">
          <AddAppButton />
          <p className="text-sky-400 flex items-center gap-2">
            <IconSpinner className="fill-sky-400 text-slate-800" />
            Auto-detecting Apps
          </p>
        </div>
      </header>
      <div className="flex items-center gap-2 py-6">
        <IconFunction />
        <p className="text-white">{apps.length} Apps Connected</p>
      </div>
      <div className="grid md:grid-cols-2 grid-cols-1 gap-6 min-h-max">
        {/* To do: fetch real apps */}
        {apps.map((app) => {
          return <AppCard key={app?.id} app={app} />;
        })}
      </div>
    </div>
  );
}
