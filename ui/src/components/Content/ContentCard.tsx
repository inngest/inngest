import classNames from '../../utils/classnames'

export default function ContentCard({
  children,
  title,
  datetime,
  button,
  id,
  active,
}) {
  console.log(active)

  return (
    <div
      className={classNames(
        active ? `bg-slate-950` : ``,
        `flex-1 border rounded-lg border-slate-800/50`
      )}
    >
      <div className="px-5 pt-3.5 ">
        <div className="mb-5">
          <h1 className="text-base text-slate-50">{title}</h1>
          <span className="text-2xs mt-1 block">{datetime}</span>
        </div>

        <div className="flex items-center justify-between border-b border-slate-800/50 pb-5">
          {button && button}
          <span className="text-3xs leading-none">{id}</span>
        </div>
      </div>
      <div>{children}</div>
    </div>
  )
}
