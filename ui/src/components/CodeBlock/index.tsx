import Button from '../Button'

export default function CodeBlock() {
  return (
    <div className="w-full bg-slate-800/30 border border-slate-700/30 rounded-lg shadow ">
      <div className="bg-slate-800/40 flex justify-between">
        <div className="flex">
          <a
            className="text-xs px-5 py-2.5 border-b-2 border-indigo-400 text-white block"
            href="#"
          >
            Payload
          </a>
          <a
            className="text-xs px-5 py-2.5 border-b-2 border-transparent text-slate-400 block hover:text-slate-50 transition-all"
            href="#"
          >
            Schema
          </a>
        </div>
        <div className="flex gap-2 items-center mr-2">
          <Button label="Copy" />
          <Button label="Expand" />
        </div>
      </div>
      <div className="overflow-hidden">
        <code className="">
          <pre className="p-4 overflow-x-scroll text-2xs">
            {`{
  name: 'some.scope/event.name',
  data: {
    // Each event will always have a "data" object, super long line designed to break outside of its parent container
    // this can be a few fields
    // or a few hundred fields
    fields: { can: 'be nested objects' },
  },
  // optional
  user: {
    id: '1234567890',
  },
  ts: 1667221378334, // This will be in every event
  v: '2022-10-31.1', // optional
 }`}
          </pre>
        </code>
      </div>
    </div>
  )
}
