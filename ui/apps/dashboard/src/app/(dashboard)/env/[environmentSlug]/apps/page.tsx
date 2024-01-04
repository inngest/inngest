import Squares2X2Icon from '@heroicons/react/20/solid/Squares2X2Icon';

import Header from '@/components/Header/Header';
import { Apps } from './Apps';

export default function Page() {
  return (
    <>
      <Header title="Apps" icon={<Squares2X2Icon className="h-5 w-5 text-white" />} />
      <div className="h-full overflow-y-auto bg-slate-100">
        <Apps />
      </div>
    </>
  );
}
