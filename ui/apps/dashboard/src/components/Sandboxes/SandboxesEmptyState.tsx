import { useEffect, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';

import {
  FEATURES,
  INTRO_LEAD,
  INTRO_REST,
  USE_CASES,
} from './sandboxesContent';
import { trackSandboxesViewed, trackSandboxWaitlistJoined } from './tracking';
import WaitlistModal from './WaitlistModal';

export default function SandboxesEmptyState() {
  const [modalOpen, setModalOpen] = useState(false);
  const hasTrackedViewed = useRef(false);

  // Fire the feature-tagged page-view event once per mount.
  useEffect(() => {
    if (hasTrackedViewed.current) return;
    hasTrackedViewed.current = true;
    trackSandboxesViewed();
  }, []);

  function openModal() {
    trackSandboxWaitlistJoined();
    setModalOpen(true);
  }

  return (
    <div className="mx-auto w-full max-w-[816px] px-6 py-16">
      <h1 className="text-basis text-2xl">Sandboxes</h1>
      <p className="text-subtle mt-2 text-sm leading-relaxed">
        <span className="text-basis font-medium">{INTRO_LEAD}</span>
        {INTRO_REST}
      </p>

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
        {FEATURES.map(({ Icon, title, description }) => (
          <div
            key={title}
            className="border-subtle bg-canvasBase rounded-md border p-3"
          >
            <div className="text-muted flex h-9 w-9 items-center justify-center">
              <Icon className="h-6 w-6" />
            </div>
            <h3 className="text-basis mt-3 text-base">{title}</h3>
            <p className="text-muted mt-1 text-xs leading-relaxed">
              {description}
            </p>
          </div>
        ))}
      </div>

      <h2 className="text-basis mt-10 text-base">What can you build?</h2>
      <div className="mt-4 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2">
        {USE_CASES.map(({ Icon, title, description }) => (
          <div key={title} className="flex items-start gap-3">
            <div className="border-subtle text-muted flex h-[46px] w-[46px] shrink-0 items-center justify-center rounded-md border">
              <Icon className="h-6 w-6" />
            </div>
            <div>
              <h3 className="text-basis text-base">{title}</h3>
              <p className="text-muted mt-0.5 text-xs">{description}</p>
            </div>
          </div>
        ))}
      </div>

      <div className="mt-10">
        <Button kind="primary" label="Join the waitlist" onClick={openModal} />
      </div>

      <WaitlistModal isOpen={modalOpen} onClose={() => setModalOpen(false)} />
    </div>
  );
}
