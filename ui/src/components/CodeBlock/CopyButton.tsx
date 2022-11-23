import { useState } from 'preact/hooks'
import classNames from '../../utils/classnames'

export default function CopyButton({ btnAction }) {
  const [clickedState, setClickedState] = useState(false)

  const handleClick = () => {
    setClickedState(true)
    btnAction()
    setTimeout(() => {
      setClickedState(false)
    }, 1000)
  }

  return (
    <button
      onClick={() => handleClick()}
      className={classNames(
        clickedState
          ? `bg-emerald-500 border-transparent`
          : `bg-slate-700/50 hover:bg-slate-700/80 border-slate-700/50`,
        `flex gap-1.5 items-center  border text-xs  rounded-sm px-2.5 py-1 text-slate-100 transition-all duration-150`
      )}
    >
      {clickedState ? 'Copied!' : 'Copy'}
    </button>
  )
}
