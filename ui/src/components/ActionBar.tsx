export default function ActionBar({ tabs }) {
  return (
    <div className="col-span-2 row-start-2 col-start-2 bg-slate-950/50 relative z-50 backdrop-blur-md border-b border-slate-800">
      <div className="flex h-full">{tabs}</div>
    </div>
  )
}
