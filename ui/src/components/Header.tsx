import { InngestLogo } from "../icons";

type Props = {
  children?: React.ReactNode;
};

export default function Header(props: Props) {
  return (
    <header className="flex w-full justify-between bg-slate-950 pr-5 pl-6 py-4 border-b border-slate-800/30 col-span-3">
      <nav className="flex items-center gap-3">
        <h1 className="text-slate-300 text-sm flex items-end ">
          <InngestLogo />
          <span className="ml-1.5 text-indigo-400">Dev Server</span>
        </h1>
        {props.children}
        {/* <span className="flex bg-slate-800 text-xs text-slate-300 items-center rounded px-2 py-1.5 ml-5 leading-none">
          <span className="bg-lime-400 w-2 h-2 rounded-full block mr-1.5">
            {''}
          </span>
          localhost:3000
        </span> */}
      </nav>

      {/* <button className="text-slate-300 text-xs">ed+inngest@edpoole.me</button> */}
    </header>
  );
}
