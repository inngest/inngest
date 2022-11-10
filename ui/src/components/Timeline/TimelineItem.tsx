import classNames from '../../utils/classnames'
import statusStyles from '../../utils/statusStyles'

export default function TimelineItem({
  children,
  status,
  active,
  topLine = true,
  bottomLine = true,
  iconOffset = 20,
}) {
  const itemStatus = statusStyles(status)

  return (
    <li className="flex pr-3.5 relative group">
      <div className="basis-[36px] shrink-0 flex flex-col items-center">
        {topLine && (
          <div
            className={`w-[2px] bg-slate-700 h-full mb-2`}
            style={`flex-basis: ${iconOffset}px`}
          ></div>
        )}
        <div className="w-full flex items-center justify-center">
          <itemStatus.icon />
        </div>
        {bottomLine && <div className="w-[2px] bg-slate-700 h-full mt-2"></div>}
      </div>

      {children}
    </li>
  )
}
