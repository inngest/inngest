import classNames from '../../utils/classnames'

export default function ContentCard({
  children,
  title,
  datetime,
  button,
  id,
  active,
}) {
  return (
    <div
      className={classNames(
        active ? `bg-slate-950` : ``,
        `flex-1 border rounded-lg border-slate-800/30 overflow-hidden flex flex-col shrink-0`
      )}
    >
      <div className="shadow-slate-950 px-5 py-4 shadow-lg relative z-30">
        <div className="mb-5">
          <h1 className=" text-lg text-slate-50">{title}</h1>
          <span className="text-2xs mt-1 block">{datetime}</span>
        </div>

        <div className="flex items-center justify-between">
          {button && button}
          <span className="text-3xs leading-none">{id}</span>
        </div>
      </div>
      <div className="overflow-y-scroll">{children}</div>
    </div>
  )
}
