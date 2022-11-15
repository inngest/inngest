import { InngestLogo } from '../icons'

export default function Header() {
  return (
    <header className="flex w-full justify-between bg-slate-950 pr-5 pl-3 py-3 border-b border-slate-800/30 col-span-3">
      <div className="flex items-center">
        <h1 className="text-slate-300 text-sm flex items-center ">
          <InngestLogo />
          <span className="ml-1.5">Inngest Server</span>
        </h1>
        <span className="flex bg-slate-800 text-xs text-slate-300 items-center rounded px-2 py-1.5 ml-5 leading-none">
          <span className="bg-lime-400 w-2 h-2 rounded-full block mr-1.5">
            {''}
          </span>
          localhost:3000
        </span>
      </div>

      <button className="text-slate-300 text-xs">ed+inngest@edpoole.me</button>
    </header>
  )
}
