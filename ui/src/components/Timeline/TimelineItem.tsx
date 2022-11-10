import classNames from '../../utils/classnames'
import statusStyles from '../../utils/statusStyles'

export default function TimelineItem({ children, status, active }) {
  const itemStatus = statusStyles(status)

  return (
    <li className="flex pr-3.5 relative group py-2">
      <div className="w-[2px] bg-slate-700 absolute top-0 left-[17px] bottom-0 group-first:top-[26px] group-last:bottom-[20px]"></div>
      <div className="basis-[36px] shrink-0 flex items-end justify-center relative z-10">
        <span className="w-[24px] h-[34px] flex items-center justify-center bg-slate-950 mb-[14px]">
          <itemStatus.icon />
        </span>
      </div>

      {children}
    </li>
  )
}
