import React, { ReactElement } from 'react';

type SectionHeaderProps = {
  pre?: ReactElement | string;
  title: ReactElement | string;
  lede?: ReactElement | string;
  center?: boolean;
};

const SectionHeader = ({ title, lede = '', center = false, pre }: SectionHeaderProps) => {
  return (
    <>
      {pre && (
        <p
          className={`text-lg leading-5 text-indigo-400 md:leading-7 ${
            center ? 'text-center' : ''
          }`}
        >
          {pre}
        </p>
      )}
      <h2
        className={`mb-2 text-2xl font-medium tracking-tighter text-slate-50 md:mb-4 md:text-4xl xl:text-5xl ${
          center ? 'text-center' : ''
        }`}
      >
        {title}
      </h2>
      {typeof lede === 'string' ? (
        <p
          className={`max-w-md text-sm leading-5 text-slate-200 md:text-base md:leading-7 lg:max-w-xl ${
            center ? 'text-center' : ''
          }`}
        >
          {lede}
        </p>
      ) : (
        <div
          className={`max-w-md text-sm leading-5 text-slate-200 md:text-base md:leading-7 lg:max-w-xl ${
            center ? 'text-center' : ''
          }`}
        >
          {lede}
        </div>
      )}
    </>
  );
};

export default SectionHeader;
