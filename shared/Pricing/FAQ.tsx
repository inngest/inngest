export function FAQRow({ question, children }) {
  return (
    <div className="items-start justify-between py-12">
      <h3 className="mb-4 text-lg xl:text-xl font-semibold tracking-tight text-white">
        {question}
      </h3>
      <div className="flex text-sm xl:text-base flex-col gap-4 xl:leading-relaxed text-slate-200">
        {children}
      </div>
    </div>
  );
}
