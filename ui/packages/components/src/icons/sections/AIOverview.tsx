import { RiSparkling2Line } from '@remixicon/react';

// Reuses the sparkle glyph already used for AI-related UI elsewhere in the
// app (e.g. the Insights AI diagnostics banner) so the nav icon matches the
// established visual language for "AI" rather than introducing a new one.
export const AIOverviewIcon = ({ className }: { className?: string }) => (
  <RiSparkling2Line className={className} />
);
