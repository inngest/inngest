# Task exe-1233-006: Type Check, Lint, and Document Metadata Flag Flip

**Feature**: exe-1233
**Status**: pending
**Priority**: P2
**Depends on**: exe-1233-001, exe-1233-002, exe-1233-003, exe-1233-004, exe-1233-005

## Objective

Run type-checking and lint across all modified packages to verify no regressions. Document the T1-13 LaunchDarkly `enable-step-metadata` flag flip as a post-merge action in the PR description.

## Acceptance Criteria

- [ ] TypeScript type-check passes for components library: `cd ui/packages/components && pnpm type-check` (if available) or `npx tsc --noEmit`
- [ ] Lint passes for components library: `cd ui/packages/components && pnpm lint`
- [ ] All component tests pass: `cd ui/packages/components && pnpm test`
- [ ] PR description includes T1-13 post-merge action note
- [ ] No `any` types introduced
- [ ] No unused imports
- [ ] Code follows project conventions

## TDD Requirements

**Test File**: N/A (verification task, not implementation)
**Test Command**: `cd ui/packages/components && pnpm test`

## Project Tooling

**Lint**: `cd ui/packages/components && pnpm lint`
**Format**: `cd ui && pnpm run format`
**Type Check**: `cd ui/packages/components && npx tsc --noEmit` or `pnpm type-check`
**Build**: `cd ui/apps/dashboard && pnpm build`

### Verification Steps

1. **Run type check**:
   ```bash
   cd ui/packages/components && npx tsc --noEmit
   ```
   Fix any type errors introduced by tasks 001-005.

2. **Run lint**:
   ```bash
   cd ui/packages/components && pnpm lint
   ```
   Fix any lint warnings/errors.

3. **Run all component tests**:
   ```bash
   cd ui/packages/components && pnpm test
   ```
   All tests from tasks 001-005 must pass.

4. **Run format**:
   ```bash
   cd ui && pnpm run format
   ```

5. **Document T1-13 in PR description**: Add the following to the PR description:

   ```markdown
   ## Post-Merge Actions

   - [ ] **T1-13**: Flip `enable-step-metadata` LaunchDarkly flag default to `true`
     - Validate in staging first to confirm metadata tab renders correctly with production data
     - This enables the Metadata tab for all users, showing AI metadata, HTTP metadata, warnings, and userland metadata
   ```

## Implementation Notes

### T1-13 Context

The `enable-step-metadata` feature flag is already used in the code at two locations:
- `StepInfo.tsx:151`: `const { value: metadataIsEnabled } = booleanFlag('enable-step-metadata', false);`
- `TopInfo.tsx:114`: `const { value: metadataIsEnabled } = booleanFlag('enable-step-metadata', false);`

The `MetadataAttrs` component is fully implemented and works. The flag default just needs to be changed from `false` to `true` in LaunchDarkly (not in code). This is a configuration change done in the LaunchDarkly dashboard.

### Common Type Errors to Watch For

- `PillElement` may not accept `data-testid` prop (if used in task 001). Wrap in `<span>` if needed.
- `LinkElement` requires `href` prop — ensure all pathCreator calls return valid values.
- `IDElement` is imported but may not have been in the original import list — verify import statement.

### Common Lint Issues

- Unused imports from tasks that were modified
- Missing `key` props on mapped elements
- Console.info statements (may need lint-disable comments if linter flags them)

## Embedded Spec Context

<details>
<summary>Relevant Requirements</summary>

**T1-13: Enable Step Metadata Flag** (MEDIUM priority)
- No code changes — LaunchDarkly config change
- Flip `enable-step-metadata` default to `true`
- Validate in staging first
- Document as post-merge action in PR description

</details>

<details>
<summary>Architecture Context</summary>

- This is the final verification task
- Ensures all code from tasks 001-005 compiles and passes lint
- T1-13 is documented in PR description, not implemented in code

</details>

<details>
<summary>Code Patterns</summary>

- TypeScript strict mode — `npx tsc --noEmit`
- ESLint — `pnpm lint`
- Prettier — `pnpm run format`
- All tests via — `pnpm test`

</details>

## Retro Requirements

**Feature Retro File**: `specs/exe-1233/retro.md`
**Project Retro File**: `specs/retros.md`

### Before Starting (CRITICAL)

1. Read BOTH retro files for lessons from previous work

### After Implementation

If there are genuine takeaways from the overall feature, add them to `specs/exe-1233/retro.md`.
