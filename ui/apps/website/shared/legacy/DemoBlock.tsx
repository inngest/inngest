import React, { useEffect, useState } from 'react';
import styled from '@emotion/styled';
import { trackDemoView } from 'src/utils/tracking';

import Play from '../Icons/Play';
import Button from './Button';
import DemoModal from './DemoModal';

export default function DemoBlock({
  headline,
  description,
}: {
  headline: string;
  description: string;
}) {
  const [demo, setDemo] = useState(false);

  useEffect(() => {
    if (demo === true) {
      trackDemoView();
    }
  }, [demo]);

  return (
    <div className="container mx-auto max-w-5xl px-12">
      <div className="flex flex-col gap-8 border-t-2 px-2 pt-8 md:flex-row">
        <div className="basis-1/4" style={{ minWidth: '260px' }}>
          <h2 className="mb-6 text-lg font-normal">{headline}</h2>
          <Button
            kind="primary"
            size="medium"
            href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=demo-section`}
          >
            Start building
          </Button>
        </div>
        <div className="basis-3/4">
          <p className="text-md text-color-gray-purple pb-6">{description}</p>
          <VidPlaceholder
            className="flex cursor-pointer items-center"
            onClick={() => setDemo(true)}
          >
            <button className="flex items-center justify-center" onClick={() => setDemo(true)}>
              <Play outline={false} fill="var(--primary-color)" size={80} />
            </button>
            <img
              src="/assets/demo/twilio-sms-demo-preview.jpg"
              className="border-color-iris-60 rounded-md border-2"
            />
          </VidPlaceholder>
        </div>
      </div>
      <DemoModal show={demo} onClickClose={() => setDemo(false)} />
    </div>
  );
}

const VidPlaceholder = styled.div`
  position: relative;

  &:hover {
    svg {
      box-shadow: 0 0 80px 20px var(--primary-color);
    }
  }

  button {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;

    svg {
      box-shadow: 0 0 40px var(--primary-color);
      border-radius: 60px;
      transition: all 0.3s;
    }
  }
`;
