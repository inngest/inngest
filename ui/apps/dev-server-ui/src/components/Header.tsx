import AddAppButton from '@/components/App/AddAppButton';
import { InngestLogo, InngestSmallLogo } from '@/icons';

type Props = {
  children?: React.ReactNode;
};

export default function Header(props: Props) {
  return (
    <header className="bg-slate-910 col-span-3 flex w-full items-center justify-between border-b border-slate-800/30 pl-6 pr-5">
      <nav className="flex items-center gap-3">
        <h1 className="flex items-end text-sm text-slate-300">
          <InngestSmallLogo className="block md:hidden" />
          <InngestLogo className="hidden md:block" />
          <span className="ml-1.5 hidden text-indigo-400 md:block">Dev Server</span>
        </h1>
        {props.children}
      </nav>
      <div className="my-1 hidden md:block">
        <AddAppButton />
      </div>
    </header>
  );
}
