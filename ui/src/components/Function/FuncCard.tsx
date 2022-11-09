import statusStyles from '../../utils/statusStyles'

export default function FuncCard({
  title,
  datetime,
  badge,
  id,
  status,
  contextualButton,
}) {
  const itemStatus = statusStyles(status)

  const contextualBar =
    status === 'PAUSED' || status === 'FAILED' ? true : false

  return (
    <div className="px-5 py-3.5 bg-slate-800/50 w-full mb-3 rounded-lg hover:bg-slate-800/80">
      <a href="#">
        <div className="flex items-start justify-between">
          <div>
            <span className="text-2xs mt-1 block leading-none">{datetime}</span>
            <h1 className="text-white mt-2">{title}</h1>
          </div>
          {badge && <div className="flex items-center">{badge}</div>}
        </div>
        <div className="flex items-center justify-between mt-2">
          <span className="text-3xs leading-none">{id}</span>
          <span className="text-3xs leading-none flex items-center">
            <itemStatus.icon />
            <span className="ml-2">{status}</span>
          </span>
        </div>
      </a>

      {contextualBar && (
        <div className="border-t border-slate-700/50 mt-5 pt-3 flex items-center justify-between">
          <p>Function paused for sleep until 1:40pm</p>
          {contextualButton && contextualButton}
        </div>
      )}
    </div>
  )
}
