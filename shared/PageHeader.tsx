export default function PageHeader({ title, lede }) {
  return (
    <div className="max-w-2xl ">
      <h1 className="text-4xl leading-[48px] sm:text-5xl sm:leading-[58px] lg:text-6xl font-semibold lg:leading-[68px] tracking-[-2px] text-slate-50 mb-5">
        {title}
      </h1>
      <p className="text-sm md:text-base text-slate-200 max-w-xl leading-6 md:leading-7">
        {lede}
      </p>
    </div>
  );
}
