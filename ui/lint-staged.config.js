const path = require('path');

const ESLintTask = (fileNames) => {
  const filesByWorkspace = {};

  for (const fileName of fileNames) {
    const relativePath = path.relative(process.cwd(), fileName);
    const parts = relativePath.split(path.sep);

    //
    // Group files by workspace (apps/* or packages/*)
    if (parts.length >= 2 && (parts[0] === 'apps' || parts[0] === 'packages')) {
      const workspace = path.join(parts[0], parts[1]);
      const workspaceRelativePath = parts.slice(2).join(path.sep);

      if (!filesByWorkspace[workspace]) {
        filesByWorkspace[workspace] = [];
      }
      filesByWorkspace[workspace].push(workspaceRelativePath);
    }
  }

  return Object.entries(filesByWorkspace).map(
    ([workspace, files]) => `cd ${workspace} && eslint --fix ${files.join(' ')}`
  );
};

module.exports = {
  //
  // Run ESLint and Prettier for TypeScript files in apps and packages
  './{apps,packages}/*/{src,test}/**/*.{tsx,ts}': [ESLintTask, 'prettier --write'],

  //
  // Run Prettier for non-TypeScript files, excluding config files
  '!(./{src,test}/**/*.{tsx,ts}|**/.prettierrc*|**/.eslintrc*|**/eslint.config.*|**/.prettierignore)':
    'prettier --ignore-unknown --write',
};
