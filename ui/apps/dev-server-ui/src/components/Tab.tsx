import classNames from '../utils/classnames';

type TabProps = {
  key: string;
  label: string;
  active: boolean;
  tabAction: (key: string) => void;
};

export default function Tab({ key, label, active, tabAction }: TabProps) {
  return (
    <button
      className={classNames(
        active ? `border-indigo-400 text-white` : `border-transparent text-slate-400`,
        `block border-b-2 px-5 py-2.5 text-xs transition-all duration-150`
      )}
      onClick={() => tabAction(key)}
    >
      {label}
    </button>
  );
}
