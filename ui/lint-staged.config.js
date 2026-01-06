const path = require('path');

//
// ESLint task that runs eslint via pnpm in the correct workspace
const ESLintTask = (fileNames) => {
  const filesByApp = {};

  fileNames.forEach((fileName) => {
    const relativePath = path.relative(process.cwd(), fileName);
    const match = relativePath.match(/^apps\/([^/]+)\//);

    if (match) {
      const app = match[1];
      if (!filesByApp[app]) {
        filesByApp[app] = [];
      }
      const appRelativePath = relativePath.replace(`apps/${app}/`, '');
      filesByApp[app].push(appRelativePath);
    }
  });

  return Object.entries(filesByApp).map(
    ([app, files]) => `cd apps/${app} && pnpm eslint --fix ${files.join(' ')}`
  );
};

module.exports = {
  //
  // Run ESLint and Prettier consecutively for TypeScript files in apps
  './apps/*/{src,test}/**/*.{tsx,ts}': [ESLintTask, 'prettier --write'],
  //
  // Run Prettier for non-TypeScript files, excluding config files
  '!(./{src,test}/**/*.{tsx,ts}|**/.prettierrc*|**/.eslintrc*|**/eslint.config.*|**/.prettierignore)':
    'prettier --ignore-unknown --write',
};
