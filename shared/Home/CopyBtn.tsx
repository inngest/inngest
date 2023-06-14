import classNames from "src/utils/classNames";
import { useState } from "react";

export default function CopyBtn({ btnAction, copy }) {
  const [clickedState, setClickedState] = useState(false);

  const handleClick = (copy) => {
    setClickedState(true);
    btnAction(copy);
    setTimeout(() => {
      setClickedState(false);
    }, 1000);
  };

  return (
    <button className="ml-1 relative group" onClick={() => handleClick(copy)}>
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
        className={classNames(
          clickedState
            ? `text-indigo-400`
            : `text-slate-400 group-hover:text-white`,
          `transition-all duration-150 w-4 h-4`
        )}
      >
        <path
          fillRule="evenodd"
          d="M13.887 3.182c.396.037.79.08 1.183.128C16.194 3.45 17 4.414 17 5.517V16.75A2.25 2.25 0 0114.75 19h-9.5A2.25 2.25 0 013 16.75V5.517c0-1.103.806-2.068 1.93-2.207.393-.048.787-.09 1.183-.128A3.001 3.001 0 019 1h2c1.373 0 2.531.923 2.887 2.182zM7.5 4A1.5 1.5 0 019 2.5h2A1.5 1.5 0 0112.5 4v.5h-5V4z"
          clipRule="evenodd"
        />
      </svg>
      <div className="absolute opacity-0 group-hover:opacity-100 transition-all duration-150 bg-slate-900/80 text-slate-300 font-semibold text-xs px-3 py-1.5 rounded bottom-[30px] -left-[20px]">
        {clickedState ? "Copied" : "Copy"}
      </div>
    </button>
  );
}
