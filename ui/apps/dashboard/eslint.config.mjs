//  @ts-check

import { tanstackConfig } from '@tanstack/eslint-config';

export default [
  ...tanstackConfig,
  {
    //
    // TODO: Remove these overrides once migration is done
    rules: {
      'import/consistent-type-specifier-style': 'off',
      'import/order': 'off',
      'sort-imports': 'off',
      '@typescript-eslint/ban-ts-comment': 'off',
      '@stylistic/spaced-comment': 'off',
      'no-shadow': 'off',
      '@typescript-eslint/no-unnecessary-condition': 'off',
      '@typescript-eslint/array-type': 'off',
      '@typescript-eslint/require-await': 'off',
      '@typescript-eslint/consistent-type-imports': 'off',
      'prefer-const': 'off',
      'no-empty-pattern': 'off',
      'import/newline-after-import': 'off',
      '@typescript-eslint/naming-convention': 'off',
      'node/prefer-node-protocol': 'off',
    },
  },
  {
    ignores: ['src/gql/gql.ts', 'src/gql/graphql.ts'],
  },
];
