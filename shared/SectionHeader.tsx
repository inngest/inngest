import React, { ReactElement } from "react";

type SectionHeaderProps = {
  pre?: ReactElement | string;
  title: ReactElement | string;
  lede?: ReactElement | string;
  center?: boolean
};

const SectionHeader = ({ title, lede = "", center = false, pre }: SectionHeaderProps) => {
  return (
    <>
      {pre && (
        <p className={`text-indigo-400 text-lg leading-5 md:leading-7 ${center ? "text-center" : "" }`}>
          {pre}
        </p>
      )}
      <h2 className={`text-slate-50 font-medium text-2xl md:text-4xl xl:text-5xl mb-2 md:mb-4 tracking-tighter ${center ? "text-center" : "" }`}>
        {title}
      </h2>
      {typeof lede === "string" ? (
        <p className={`text-slate-200 max-w-md lg:max-w-xl text-sm md:text-base leading-5 md:leading-7 ${center ? "text-center" : "" }`}>
          {lede}
        </p>
      ) : (
        <div className={`text-slate-200 max-w-md lg:max-w-xl text-sm md:text-base leading-5 md:leading-7 ${center ? "text-center" : "" }`}>
          {lede}
        </div>
      )}
    </>
  );
};

export default SectionHeader;
