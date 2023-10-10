import classNames from '../utils/classnames';

export default function Tab({ key, label, active, tabAction }) {
  return (
    <button
      className={classNames(
        active ? `border-indigo-400 text-white` : `border-transparent text-slate-400`,
        `text-xs px-5 py-2.5 border-b-2 block transition-all duration-150`,
      )}
      onClick={() => tabAction(key)}
    >
      {label}
    </button>
  );
}
