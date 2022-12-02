import React, { ReactElement } from "react";

type SectionHeaderProps = {
  title: ReactElement | string;
  lede: ReactElement | string;
};

const SectionHeader = ({ title, lede }: SectionHeaderProps) => {
  return (
    <>
      <h2 className="text-slate-50 font-medium text-2xl md:text-4xl xl:text-5xl mb-2 md:mb-4 tracking-tighter ">
        {title}
      </h2>
      {typeof lede === "string" ? (
        <p className="text-slate-400 max-w-md lg:max-w-xl text-sm md:text-base leading-5 md:leading-7">
          {lede}
        </p>
      ) : (
        <div className="text-slate-400 max-w-md lg:max-w-xl text-sm md:text-base leading-5 md:leading-7">
          {lede}
        </div>
      )}
    </>
  );
};

export default SectionHeader;
