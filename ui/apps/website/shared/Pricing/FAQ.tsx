export function FAQRow({ question, children }) {
  return (
    <div className="items-start justify-between py-6 lg:py-12">
      <h3 className="mb-4 text-lg font-semibold tracking-tight text-white xl:text-xl">
        {question}
      </h3>
      <div className="prose text-sm text-slate-200 xl:text-base xl:leading-relaxed">{children}</div>
    </div>
  );
}
