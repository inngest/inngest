const path = require('path');

// Single-quote a path for bash so shell metacharacters (e.g. `$` in
// TanStack Router's `$envSlug` filenames) aren't expanded. The outer
// command string is wrapped in double quotes, which lint-staged's
// command parser (string-argv) handles correctly.
const shellQuote = (p) => `'${p.replace(/'/g, `'\\''`)}'`;

// lint-staged v15 runs command strings via execa without a shell, so
// `&&` and single-quoted filenames need an explicit shell invocation.
const wrapInShell = (command) => `bash -c "${command}"`;

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

  return Object.entries(filesByWorkspace).map(([workspace, files]) =>
    wrapInShell(`cd ${shellQuote(workspace)} && eslint --fix ${files.map(shellQuote).join(' ')}`)
  );
};

const PrettierTask = (fileNames) =>
  wrapInShell(`prettier --write ${fileNames.map(shellQuote).join(' ')}`);

const PrettierIgnoreUnknownTask = (fileNames) =>
  wrapInShell(`prettier --ignore-unknown --write ${fileNames.map(shellQuote).join(' ')}`);

module.exports = {
  //
  // Run ESLint and Prettier for TypeScript files in apps and packages
  './{apps,packages}/*/{src,test}/**/*.{tsx,ts}': [ESLintTask, PrettierTask],

  //
  // Run Prettier for non-TypeScript files, excluding config files
  '!(./{src,test}/**/*.{tsx,ts}|**/.prettierrc*|**/.eslintrc*|**/eslint.config.*|**/.prettierignore)':
    PrettierIgnoreUnknownTask,
};
