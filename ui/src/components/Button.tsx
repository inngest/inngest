export default function Button({ label, icon }) {
  return (
    <button className="flex gap-1.5 items-center bg-slate-700/50 border text-xs border-slate-700/50 rounded-sm px-2.5 py-1 text-slate-100 hover:bg-slate-700/80 transition-all duration-150">
      {label && label}
      {icon && icon}
    </button>
  )
}
