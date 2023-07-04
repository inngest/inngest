export default function ContentFrame({ children }) {
  return (
    <main className="flex flex-1 overflow-hidden row-start-3 col-start-2">
      <div className="flex gap-3 p-3 w-full min-w-0">{children}</div>
    </main>
  )
}
