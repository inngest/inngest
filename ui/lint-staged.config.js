const path = require('path');

const ESLintTask = (fileNames) =>
  `next lint --format pretty --fix --file ${fileNames
    .map((f) => path.relative(process.cwd(), f))
    .join(' --file ')}`;

module.exports = {
  // Run ESLint and Prettier consecutively for TypeScript files
  './{src,test}/**/*.{tsx,ts}': [ESLintTask, 'prettier --write'],
  // Run Prettier for non-TypeScript files
  '!(./{src,test}/**/*.{tsx,ts})': 'prettier --ignore-unknown --write',
};
