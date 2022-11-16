export default function TimelineStaticRow({ label, datetime, actionBtn }) {
  return (
    <div className="flex items-start justify-between w-full pt-[2px]">
      <div>
        <h2 className="text-slate-50">{label}</h2>
        {datetime && (
          <span className="text-2xs mt-1 block leading-none text-slate-400">
            {datetime}
          </span>
        )}
      </div>
      {actionBtn && actionBtn}
    </div>
  )
}
