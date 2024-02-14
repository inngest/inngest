export function FAQRow({ question, children }) {
  return (
    <div className="items-start justify-between py-6 lg:py-12">
      <h3 className="mb-4 text-lg xl:text-xl font-semibold tracking-tight text-white">
        {question}
      </h3>
      <div className="text-sm xl:text-base xl:leading-relaxed text-slate-200 prose">
        {children}
      </div>
    </div>
  );
}
