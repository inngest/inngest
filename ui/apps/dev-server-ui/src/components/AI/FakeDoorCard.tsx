import { useState } from 'react';

import { Button } from '@inngest/components/Button';

import {
  useFakeDoorTracking,
  type FakeDoorFeature,
} from './useFakeDoorTracking';

const COPY: Record<
  FakeDoorFeature,
  {
    title: string;
    description: string;
    cta: string;
    thanksTitle: string;
    thanksDescription: string;
  }
> = {
  scores: {
    title: 'See your scores in Inngest Cloud once you deploy',
    description:
      "The Dev Server doesn't display scores yet. Want them here too?",
    cta: 'Yes, I want this',
    thanksTitle: "Thanks, we've heard you!",
    thanksDescription:
      'Your feedback helps us decide whether to bring scores to the Dev Server. Until then, deploy your app to see them in Cloud.',
  },
  experiments: {
    title: 'See your experiments in Inngest Cloud once you deploy',
    description:
      "The Dev Server doesn't display experiments yet. Want them here too?",
    cta: 'Yes, I want this',
    thanksTitle: "Thanks, we've heard you!",
    thanksDescription:
      'Your feedback helps us decide whether to bring experiments to the Dev Server. Until then, deploy your app to see them in Cloud.',
  },
};

const storageKey = (feature: FakeDoorFeature) =>
  `inngest:fakeDoor:${feature}:ctaClicked`;

// Fake-door banner shown on the Scores/Experiments pages above the
// onboarding content. It leads with where the data actually lives (Cloud,
// after deploying); the CTA only records interest in local support. The
// acknowledged state persists in localStorage so the same user isn't
// re-asked on every visit. Its view needs no separate tracking — it equals
// the page's own `viewed` event.
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

  const copy = COPY[feature];

  return (
    <div className="border-subtle bg-canvasBase flex flex-col items-start gap-4 rounded-md border p-6 shadow-sm">
      {clicked ? (
        <div className="flex flex-col gap-1">
          <h2 className="text-basis text-base font-medium">
            {copy.thanksTitle}
          </h2>
          <p className="text-muted text-sm leading-relaxed">
            {copy.thanksDescription}
          </p>
        </div>
      ) : (
        <>
          <div className="flex flex-col gap-1">
            <h2 className="text-basis text-base font-medium">{copy.title}</h2>
            <p className="text-muted text-sm leading-relaxed">
              {copy.description}
            </p>
          </div>
          <Button
            kind="primary"
            appearance="solid"
            label={copy.cta}
            onClick={onClick}
          />
        </>
      )}
    </div>
  );
}
