import AddAppButton from '@/components/App/AddAppButton';
import { InngestLogo, InngestSmallLogo } from '@/icons';

type Props = {
  children?: React.ReactNode;
};

export default function Header(props: Props) {
  return (
    <header className="flex w-full items-center justify-between bg-slate-910 pr-5 pl-6 border-b border-slate-800/30 col-span-3">
      <nav className="flex items-center gap-3">
        <h1 className="text-slate-300 text-sm flex items-end">
          <InngestSmallLogo className="block md:hidden" />
          <InngestLogo className="hidden md:block" />
          <span className="ml-1.5 text-indigo-400 hidden md:block">Dev Server</span>
        </h1>
        {props.children}
      </nav>
      <div className="my-1 md:block hidden">
        <AddAppButton />
      </div>
    </header>
  );
}
