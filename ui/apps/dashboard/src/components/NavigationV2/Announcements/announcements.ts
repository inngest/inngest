// Marketing-authored announcement cards shown in a small stack at the bottom of
// the left sidebar. Add an entry here to publish an announcement; remove it (or
// let its `endDate` pass) to retire it.
//
// Guidelines:
//   - `id` must be stable and unique. It is the dismissal key, so reusing an old
//     id would resurface for users who already dismissed that earlier card, and
//     changing an id re-shows it to everyone.
//   - Keep `title`/`body` short — the card is fixed-height and body text is
//     clamped to two lines.
//   - `cta.href` is validated against an allow-list (http/https/mailto or a
//     site-relative path). Anything else is dropped.
//   - Omit `startDate`/`endDate` for "always on". Dates are ISO strings and the
//     window is inclusive on both ends.

export type Announcement = {
  /** Stable, unique id. Used as the dismissal key — never reuse or rename. */
  id: string;
  title: string;
  body: string;
  /** Optional illustration/screenshot rendered in a fixed-height slot. */
  imageUrl?: string;
  /** Optional dark-mode variant of `imageUrl`. Falls back to `imageUrl` when omitted. */
  imageUrlDark?: string;
  /** Optional call-to-action. `href` is validated before render. */
  cta?: { label: string; href: string };
  /** ISO date; the card is shown on or after this instant. Omit = always started. */
  startDate?: string;
  /** ISO date; the card is hidden after this instant. Omit = never expires. */
  endDate?: string;
};

export const announcements: Announcement[] = [
  {
    id: 'insight-queries-2026-06',
    title: 'Introducing defer ( )',
    body: 'Schedule runs when the parent run finishes.',
    imageUrl: '/images/announcements/defer-dark.png',
    imageUrlDark: '/images/announcements/defer-dark.png',
    cta: {
      label: 'Learn more',
      href: 'https://www.inngest.com/blog/announcing-defer',
    },
    startDate: '2026-06-01T00:00:00Z',
    endDate: '2026-07-01T00:00:00Z',
  },

  // {
  //   id: 'dashboards-2026-06',
  //   title: 'Announcing insights',
  //   body: 'Query runs and step data across your project',
  //   // imageUrl: '/images/announcements/defer-dark.png',
  //   // imageUrlDark: '/images/announcements/step-dark.svg',
  //   cta: { label: 'Explore dashboards', href: 'https://www.inngest.com/docs' },
  //   startDate: '2026-06-01T00:00:00Z',
  //   endDate: '2026-07-01T00:00:00Z',
  // },
];
