/**
 * SVGO is a tool that optimizes SVG files. It is used by SVGR to optimize SVG files before they are
 * transformed into React components.
 *
 * For configuration options, @see {@link https://github.com/svg/svgo/blob/main/README.md}
 */
module.exports = {
  plugins: [
    {
      name: 'preset-default',
      params: {
        overrides: {
          // Removing viewBox attribute causes CSS scaling to break.
          // See https://github.com/svg/svgo/blob/main/README.md#svg-wont-scale-when-css-is-applied-on-it
          removeViewBox: false,
        },
      },
    },
    // This makes sure that IDs inside SVG files don't collide with one another so that styles
    // applied to one don't leak into others.
    'prefixIds',
  ],
};
