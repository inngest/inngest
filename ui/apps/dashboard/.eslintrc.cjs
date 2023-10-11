module.exports = {
  extends: ['next/core-web-vitals', 'prettier', 'plugin:storybook/recommended'],
  parserOptions: {
    project: true,
    tsconfigRootDir: __dirname,
  },
  plugins: ['@typescript-eslint'],
  root: true,
  rules: {
    '@typescript-eslint/no-unnecessary-type-assertion': 'error',
  },
};
