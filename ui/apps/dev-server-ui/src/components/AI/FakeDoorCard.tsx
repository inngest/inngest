import { useState } from 'react';

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

// Fake-door banner shown on the Scores/Experiments pages above the
// onboarding content. The CTA only records interest; the acknowledged
// state persists in localStorage so the same user isn't re-asked on
// every visit. Its view needs no separate tracking — it equals the
// page's own `viewed` event.
export function FakeDoorCard({ feature }: { feature: FakeDoorFeature }) {
  const track = useFakeDoorTracking(feature);
  const [clicked, setClicked] = useState(() => {
    try {
      return localStorage.getItem(storageKey(feature)) === '1';
    } catch {
      return false;
    }
  });

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
    <div className="border-subtle bg-canvasSubtle flex flex-col items-start gap-3 rounded-md border p-4">
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
