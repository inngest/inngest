import { useEffect } from 'preact/hooks'

export default function CodeBlockModal({ children, closeModal }) {
  useEffect(() => {
    const close = (e) => {
      if (e.key === 'Escape') {
        closeModal()
      }
    }
    window.addEventListener('keydown', close)
    return () => window.removeEventListener('keydown', close)
  }, [])

  return (
    <div className="fixed inset-0 z-50 px-6 flex items-center justify-center bg-black/50 w-screen h-screen  ">
      <div className=" bg-slate-950">{children}</div>
    </div>
  )
}
