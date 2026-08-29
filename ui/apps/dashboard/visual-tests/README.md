# Metric chart visual tests

This standalone Vite fixture renders the production Recharts and ECharts
metric components without starting the dashboard. The chart-ready fixtures use
a fixed bucket axis so a representation change cannot alter tick placement or
horizontal spacing. Missing observations are absent values within that axis;
the later dense response can replace them with explicit nulls and compare
against the same baselines. Adapter unit tests separately cover conversion from
sparse API responses.

Install Chromium once, then run the comparisons:

```sh
pnpm exec playwright install chromium
pnpm test:visual
```

After reviewing an intentional visual change, update the committed baselines:

```sh
pnpm test:visual:update
```

Playwright captures individual fixed-width chart containers with a UTC
timezone, light theme, 1× device scale, and animations disabled. Failed CI
runs upload the expected, actual, and diff images as an artifact. Each failed
comparison also includes a labeled `*-blink.gif` that alternates between the
expected and actual screenshots. Creating the GIF locally requires
ImageMagick (`magick` or `convert`) on `PATH`.
