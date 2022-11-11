import { useState } from 'preact/hooks'
import classNames from '../../utils/classnames'

export default function CopyButton({ btnAction }) {
  const [clickedState, setClickedState] = useState(false)

  const handleClick = () => {
    console.log('clicked')
    setClickedState(true)
    btnAction()
  }

  const handleTransitionEnd = () => {
    setClickedState(false)
  }

  return (
    <button
      onClick={() => handleClick()}
      onTransitionEnd={() => handleTransitionEnd()}
      className={classNames(
        clickedState
          ? `bg-emerald-500 translate-x-0 border-transparent duration-1000`
          : `bg-slate-700/50 hover:bg-slate-700/80 border-slate-700/50`,
        `flex gap-1.5 items-center  border text-xs  rounded-sm px-2.5 py-1 text-slate-100  transition-transform duration-1000`
      )}
    >
      {clickedState ? 'Copied!' : 'Copy'}
    </button>
  )
}
