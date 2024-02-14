import React, { ReactElement } from 'react';

import Button from './Button';

type FeatureCalloutsProps = {
  heading: ReactElement | string;
  backgrounds?: 'alternating' | 'gray';
  features: {
    topic: string;
    title: string;
    description: ReactElement | string;
    link?: {
      href: string;
      text?: string; // default = "Learn more"
    };
    image: ReactElement | string;
  }[];
  cta?: {
    href: string;
    text: string;
  };
};

const FeatureCallouts = ({
  heading,
  backgrounds = 'alternating',
  features = [],
  cta,
}: FeatureCalloutsProps) => {
  return (
    <div className="container mx-auto my-24 max-w-5xl">
      <div className="max-xl mx-auto px-6 pb-16 text-center">
        <h2 className="text-4xl">{heading}</h2>
      </div>
      {features.map((f, i) => (
        <div
          key={`feature-${i}`}
          className="flex w-full flex-col items-center px-8 py-4 lg:flex-row lg:px-0"
        >
          <div
            className={`order-2 px-6 pb-16 pt-8 lg:w-1/2 lg:px-16 lg:py-0 lg:order-${
              i % 2 === 0 ? '1' : '2'
            }`}
          >
            <div className="text-color-iris-100 pb-2 text-xs uppercase">
              <pre>{f.topic}</pre>
            </div>
            <h3 className="pb-2">{f.title}</h3>
            <p>{f.description}</p>
          </div>
          <div
            className={`${
              backgrounds === 'gray' ? 'bg-light-gray' : `alt-bg-${i}`
            } order-1 h-[350px] w-full rounded-lg bg-orange-50 p-12 sm:h-[500px] lg:h-[400px] lg:w-1/2 xl:h-[500px] lg:order-${
              i % 2 === 0 ? '2' : '1'
            } flex items-center justify-center`}
          >
            {typeof f.image === 'string' ? (
              <img src={f.image} alt={`A graphic of ${f.title} feature`} />
            ) : (
              f.image
            )}
          </div>
        </div>
      ))}
      {cta && (
        <div className="max-xl mx-auto flex flex justify-center px-6 pt-16 text-center">
          <Button kind="outlinePrimary" href={cta.href}>
            {cta.text}
          </Button>
        </div>
      )}
    </div>
  );
};

export default FeatureCallouts;
