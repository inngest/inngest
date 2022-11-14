export default function ContentFrame({ children }) {
  return (
    <main className="flex flex-1 overflow-hidden row-start-3 col-start-3">
      {children}
    </main>
  )
}
