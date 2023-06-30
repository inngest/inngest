import AppCard from '@/components/App/AppCard';
import { IconFunction, IconPlus, IconSpinner } from '@/icons';
import AddAppButton from '@/components/App/AddAppButton';

const mockApps = [
  {
    name: '',
    id: 'tempId',
    createdAt: '',
    url: 'localhost:3001',
    functionCount: 0,
    sdkVersion: '',
    status: 'not connected',
    automaticallyAdded: false,
    connecting: true,
  },
  {
    name: 'SDK Example Redwoodjs Vercel',
    id: 'id1',
    createdAt: '',
    url: 'localhost:3000',
    functionCount: 24,
    sdkVersion: '2.0.41',
    status: 'connected',
    automaticallyAdded: false,
  },
  {
    name: 'SDK Example',
    id: 'id3',
    createdAt: '',
    url: 'localhost:4000',
    functionCount: 0,
    sdkVersion: '2.0.41',
    status: 'connected',
    automaticallyAdded: true,
  },
];

export default function AppList() {
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
        <p className="text-white">{mockApps.length} Apps Connected</p>
      </div>
      <div className="grid md:grid-cols-2 grid-cols-1 gap-6 min-h-max">
        {mockApps.map((app, id) => {
          return <AppCard key={app?.id} app={app} />;
        })}
      </div>
    </div>
  );
}
