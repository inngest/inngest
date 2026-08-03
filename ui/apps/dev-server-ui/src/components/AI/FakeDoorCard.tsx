import { useEffect, useRef, useState } from 'react';

import { Button } from '@inngest/components/Button';

import {
  useFakeDoorTracking,
  type FakeDoorFeature,
} from './useFakeDoorTracking';

const COPY: Record<
  FakeDoorFeature,
  { title: string; description: string; cta: string }
> = {
  scores: {
    title: 'Want to see scores in the Dev Server?',
    description:
      "At the moment, scores aren't displayed in the Dev Server. However, you'll be able to view them once you deploy your app to Cloud.",
    cta: 'I want to see my scores in the Dev Server',
  },
  experiments: {
    title: 'Want to see experiments in the Dev Server?',
    description:
      "At the moment, experiments aren't displayed in the Dev Server. However, you'll be able to view them once you deploy your app to Cloud.",
    cta: 'I want to see my experiments in the Dev Server',
  },
};

const storageKey = (feature: FakeDoorFeature) =>
  `inngest:fakeDoor:${feature}:ctaClicked`;

// Fake-door card shown on the Scores/Experiments pages instead of the
// onboarding content once the dev server has seen the feature being used.
// The CTA only records interest; the acknowledged state persists in
// localStorage so the same user isn't re-asked on every visit.
export function FakeDoorCard({ feature }: { feature: FakeDoorFeature }) {
  const track = useFakeDoorTracking(feature);
  const [clicked, setClicked] = useState(() => {
    try {
      return localStorage.getItem(storageKey(feature)) === '1';
    } catch {
      return false;
    }
  });

  // Fire once on view (ref-guarded against StrictMode double-invoke). This
  // fires from the card's own mount, not the page view, so the detected
  // cohort is counted even when /dev resolves after the page has rendered.
  const viewedRef = useRef(false);
  useEffect(() => {
    if (viewedRef.current) return;
    viewedRef.current = true;
    track('detected_viewed');
  }, [track]);

  const onClick = () => {
    track('cta_clicked');
    try {
      localStorage.setItem(storageKey(feature), '1');
    } catch {
      // localStorage unavailable; the acknowledged state just won't persist.
    }
    setClicked(true);
  };

  return (
    <div className="border-subtle bg-canvasBase flex flex-col items-start gap-3 rounded-md border p-6">
      {clicked ? (
        <>
          <h2 className="text-basis text-base font-medium">
            Thanks — we&apos;ve heard you!
          </h2>
          <p className="text-muted text-sm leading-relaxed">
            We&apos;re using this feedback to decide whether to bring {feature}{' '}
            to the Dev Server. In the meantime, {feature} will show up in your
            Inngest Cloud dashboard once you deploy your app.
          </p>
        </>
      ) : (
        <>
          <h2 className="text-basis text-base font-medium">
            {COPY[feature].title}
          </h2>
          <p className="text-muted text-sm leading-relaxed">
            {COPY[feature].description}
          </p>
          <Button
            kind="primary"
            appearance="solid"
            label={COPY[feature].cta}
            onClick={onClick}
          />
        </>
      )}
    </div>
  );
}
