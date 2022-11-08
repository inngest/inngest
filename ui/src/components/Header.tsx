import Logo from '../icons/Logo'

export default function Header() {
  return (
    <header className="flex w-full justify-between bg-slate-950 px-5 py-3 border-b border-slate-800 fixed top-0 left-0 right-0">
      <div className="flex items-center">
        <h1 className="text-slate-200 text-sm flex items-center ">
          <Logo />
          <span className="ml-1.5">Inngest Server</span>
        </h1>
        <span className="flex bg-slate-800 text-xs text-slate-300 items-center rounded-lg px-2.5 py-1.5 ml-5">
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
