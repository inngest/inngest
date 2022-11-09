import classNames from '../utils/classNames'

export default function SidebarLink({ icon, active, badge }) {
  return (
    <button
      className={classNames(
        active
          ? `border-indigo-400`
          : `border-transparent opacity-40 hover:opacity-100`,
        `border-l-2 flex items-center justify-center w-full py-3 transition-all duration-150`
      )}
    >
      {icon}
    </button>
  )
}
