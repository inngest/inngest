import { fixupPluginRules } from '@eslint/compat';
import pluginJs from '@eslint/js';
import pluginReact from 'eslint-plugin-react';
import pluginReactHooks from 'eslint-plugin-react-hooks';
import globals from 'globals';
import tseslint from 'typescript-eslint';

// console.log(pluginReactHooks.configs.recommended)
// console.log("---")
// console.log(pluginReact.configs.flat.recommended)

export default [
  { files: ['**/*.{js,mjs,cjs,ts,jsx,tsx}'] },
  { languageOptions: { globals: globals.browser } },
  pluginJs.configs.recommended,
  ...tseslint.configs.recommended,
  pluginReact.configs.flat.recommended,
  // pluginReactHooks.configs.recommended,
  {
    plugins: {
      'react-hooks': fixupPluginRules(pluginReactHooks),
    },
  },
  {
    rules: {
      ...pluginReactHooks.configs.recommended.rules,
      'react/react-in-jsx-scope': 'off',
    },
  },

  // TODO: All of these overrides should eventually be removed. They're good
  // rules but we're not ready for them yet
  {
    rules: {
      'react-hooks/rules-of-hooks': 'off',
      '@typescript-eslint/ban-ts-comment': 'off',
      '@typescript-eslint/no-empty-object-type': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-non-null-asserted-optional-chain': 'off',
      '@typescript-eslint/no-unused-expressions': 'off',
      '@typescript-eslint/no-unused-vars': 'off',
      'no-empty': 'off',
      'no-extra-boolean-cast': 'off',
      'no-useless-escape': 'off',
      'prefer-const': 'off',
      'react/display-name': 'off',
      'react/jsx-key': 'off',
      'react/jsx-no-target-blank': 'off',
      'react/no-unescaped-entities': 'off',
      'react/prop-types': 'off',

      // TODO: When enabling this, explicitly make it error. It warns by default
      'react-hooks/exhaustive-deps': 'off',
    },
  },
];
