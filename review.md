# Code Review: EXE-1681 API keys management UI (PR #4050)

## 1. Summary

This PR adds a `/settings/api-keys` route to the cloud dashboard with list/create/rename/delete flows for user-facing API keys, plus a one-time plaintext reveal step. It reuses existing `@inngest/components` primitives (Modal, AlertModal, Select, Table, Alert, CopyButton) and the existing urql setup. The code is well-structured, thoughtfully commented where the logic is non-obvious (the render-loop and empty→populated state lifts are called out), and security-conscious around plaintext handling. The main concerns are a race condition in the create flow that can leak a stale plaintext to a subsequent modal open, stale `error` state on the delete modal across open/close cycles, unused fields over-fetched in the list query, and the complete absence of tests for a mutation-heavy feature.

## 2. Critical Issues 🔴

- [ ] No critical issues found.

## 3. Significant Concerns 🟡

- [x] **Race condition in create flow** — [CreateAPIKeyModal.tsx:76-92](ui/apps/dashboard/src/components/APIKeys/CreateAPIKeyModal.tsx#L76-L92)
  - **Description**: If the user clicks Cancel / Esc / backdrop while the create mutation is in flight, `close()` resets `plaintextKey` to `null`, but the still-awaiting `setPlaintextKey(pt)` then fires *after* the modal is closed. The next time the modal opens, it renders the reveal step for a key the user never saw — and the plaintext sits in React state, retained in memory, tied to a key that may or may not be the one shown in the list.
  - **Impact**: Subtle UX bug (wrong reveal on reopen) plus a plaintext-retention issue that contradicts the PR's stated security posture ("modal drops plaintext when user closes the modal").
  - **Suggestion**: Gate the post-mutation `setPlaintextKey` on the modal still being open, or reset state on the `isOpen` true-edge the same way `RenameAPIKeyModal` does, or use a cancelled-ref.
    ```tsx
    // Option A: guard with a ref
    const cancelledRef = useRef(false);
    function close() {
      cancelledRef.current = true;
      // ...existing resets
      onClose();
    }
    async function submit() {
      cancelledRef.current = false;
      // ...
      if (!cancelledRef.current) setPlaintextKey(pt);
    }

    // Option B: mirror RenameAPIKeyModal's pattern
    useEffect(() => {
      if (isOpen) {
        setName(''); setSelectedEnv(null);
        setPlaintextKey(null); setError(null);
      }
    }, [isOpen]);
    ```

- [x] **Stale `error` state in DeleteAPIKeyModal** — [DeleteAPIKeyModal.tsx:23](ui/apps/dashboard/src/components/APIKeys/DeleteAPIKeyModal.tsx#L23)
  - **Description**: `error` and `isSubmitting` are never reset when the modal reopens for a different key. If a user attempts to delete key A, hits a server error, closes the modal, and opens it for key B, they'll see key A's error banner inside the "Delete key B?" confirmation.
  - **Impact**: Confusing/incorrect error messaging; a user might think their second delete failed when they haven't clicked Delete yet.
  - **Suggestion**: Add a `useEffect` keyed on `isOpen` (or `keyID`) that clears `error` when the modal opens, matching `RenameAPIKeyModal.tsx:38-43`.

- [x] **Over-fetching unused `scopes` field** — [useAPIKeys.ts:17-21](ui/apps/dashboard/src/components/APIKeys/useAPIKeys.ts#L17-L21)
  - **Description**: The query fetches `scopes { name allow deny }` on every API key, but `scopes` is not used anywhere in the UI (grep confirms `APIKeysTable`/route/page don't reference it).
  - **Impact**: Wasted payload and server work on every list render. More importantly, it implicitly commits to shipping scope data from the backend even if the feature isn't live yet, and every additional key multiplies the over-fetch.
  - **Suggestion**: Drop `scopes` from `GetAPIKeys` until the UI actually surfaces it. Regenerate `gql.ts`/`graphql.ts` afterwards.

- [x] **Raw server error messages surfaced to users** — [CreateAPIKeyModal.tsx:81](ui/apps/dashboard/src/components/APIKeys/CreateAPIKeyModal.tsx#L81), [DeleteAPIKeyModal.tsx:34](ui/apps/dashboard/src/components/APIKeys/DeleteAPIKeyModal.tsx#L34), [RenameAPIKeyModal.tsx:65](ui/apps/dashboard/src/components/APIKeys/RenameAPIKeyModal.tsx#L65)
  - **Description**: Raw `res.error.message` from urql is rendered verbatim to the user. urql `CombinedError.message` often contains prefixes like `[GraphQL] …` and internal GraphQL path info.
  - **Impact**: Poor UX for expected errors (e.g., duplicate name, permission denied) and potential info disclosure from network/transport errors.
  - **Suggestion**: Extract `res.error.graphQLErrors?.[0]?.message` (fallback to a friendly message for network errors) rather than the concatenated `message`, and consider mapping known error codes to user-facing copy.

- [x] **No test coverage** — Entire `src/components/APIKeys/` directory
  - **Description**: No tests. This is a mutation-heavy feature that modifies credentials.
  - **Impact**: Regressions to the reveal flow, cache invalidation, or validation won't be caught by CI. Any of the state-handling bugs above would be caught by a basic test of "open → submit → close → reopen".
  - **Suggestion**: At minimum, add tests for (a) modal state reset on open, (b) form validation messages, (c) `additionalTypenames` wiring, (d) reveal step renders exactly once per creation. The dashboard app doesn't seem to have many component tests today — if adopting Vitest here is out of scope, call that out in the PR description.

- [x] **No default env / no grouping in env picker** — [CreateAPIKeyModal.tsx:141](ui/apps/dashboard/src/components/APIKeys/CreateAPIKeyModal.tsx#L141)
  - **Description**: The Select `onChange` receives a single `Option` per the discriminated union, but at run time there's no guard against the type narrowing. Not an actual bug today, just a fragility note. More importantly: there is no indication of which env is the user's current/default environment — in an account with many branch children, the picker is alphabetical (whatever order `workspacesToEnvironments` returns) and the user may have to hunt for "Production".
  - **Impact**: Minor UX friction. For teams with dozens of branch envs, picking the right env is the whole interaction.
  - **Suggestion**: Default-select the production env (use `useDefaultEnvironment`) when the modal opens with no selection, and/or group envs by type in the Select (Production / Test / Branches). At minimum sort to put Production first.

## 4. Suggestions 🟢

- [x] **`CreateAPIKeyButton` is over-engineered** — [CreateAPIKeyButton.tsx:4-8](ui/apps/dashboard/src/components/APIKeys/CreateAPIKeyButton.tsx#L4-L8)
  - **Description**: `appearance` and `label` props are declared with defaults, but both call sites (`EmptyState.tsx`, `routes/.../index.tsx`) use the defaults — no caller overrides them.
  - **Impact**: Dead configurability; adds surface area with no consumer.
  - **Suggestion**: Drop the props until a second caller needs them. Or replace the whole component with an inline `<Button>`; it's a one-liner and the comment about modal-state ownership now lives in the route where it belongs.

- [x] **Generic docs link** — [index.tsx:57](ui/apps/dashboard/src/routes/_authed/settings/api-keys/index.tsx#L57), [EmptyState.tsx:30](ui/apps/dashboard/src/components/APIKeys/EmptyState.tsx#L30)
  - **Description**: Both "Learn more" and "Go to docs" link to the generic `https://www.inngest.com/docs` root.
  - **Impact**: Users looking for API key–specific docs have to hunt.
  - **Suggestion**: Point at a deeper page (e.g., `/docs/platform/api-keys`) once it exists, and add a `?ref=...` attribution tag per the repo-wide convention for internal links in the website.

- [x] **Locale-dependent timestamp with no timezone** — [APIKeysTable.tsx:49](ui/apps/dashboard/src/components/APIKeys/APIKeysTable.tsx#L49)
  - **Description**: `new Date(info.getValue()).toLocaleString()` produces locale-dependent output without an explicit time zone.
  - **Impact**: Minor — same formatting inconsistency seen elsewhere in the dashboard, but lists of credentials often need precise/comparable timestamps.
  - **Suggestion**: Use the existing date formatting helper if there is one in `@inngest/components` (this codebase has shared time utilities); at least render a relative time (e.g., "2 days ago") with an absolute timestamp on hover title.

- [x] **Empty header on actions column** — [APIKeysTable.tsx:53-79](ui/apps/dashboard/src/components/APIKeys/APIKeysTable.tsx#L53-L79)
  - **Description**: The `actions` column uses `header: ''`. The delete-icon button has `aria-label="Delete"` but the rename button relies on its visible `label`.
  - **Impact**: Screen readers announce an empty column header. Fine, but a `header: 'Actions'` with a sr-only class for the cell header would be more accessible.
  - **Suggestion**: Consider `header: () => <span className="sr-only">Actions</span>`.

- [x] **Labels not associated with inputs** — [CreateAPIKeyModal.tsx:122](ui/apps/dashboard/src/components/APIKeys/CreateAPIKeyModal.tsx#L122), [RenameAPIKeyModal.tsx:80](ui/apps/dashboard/src/components/APIKeys/RenameAPIKeyModal.tsx#L80)
  - **Description**: The input `<label>` isn't associated with the `<Input>` (no `htmlFor` / `id`).
  - **Impact**: Minor a11y — clicking the label text doesn't focus the input, and assistive tech won't pair them.
  - **Suggestion**: Add `htmlFor`/`id`, or use the Input component's built-in label prop if one exists.

- [x] **Hardcoded 128-char name limit** — [CreateAPIKeyModal.tsx:65-68](ui/apps/dashboard/src/components/APIKeys/CreateAPIKeyModal.tsx#L65-L68)
  - **Description**: Client-side 128-char limit is hardcoded; the backend presumably enforces the same limit.
  - **Impact**: If the backend changes the limit, both places need updating, and client/server can drift silently.
  - **Suggestion**: If the spec defines this as a contract, add a comment referencing the spec source. Or — better — expose the limit as a constant in one place. Minor.

- [x] **Obscure null coercion for workspaceID** — [useAPIKeys.ts:32](ui/apps/dashboard/src/components/APIKeys/useAPIKeys.ts#L32)
  - **Description**: `args.workspaceID ?? null` coerces undefined to null to make urql serialize `workspaceID: null`.
  - **Impact**: Works, but the intent is slightly obscure.
  - **Suggestion**: A one-line comment would help future readers: `// null = all workspaces in the account`.

- [ ] **Hand-typed route string with `as` cast** — [ProfileMenu.tsx:88](ui/apps/dashboard/src/components/Navigation/ProfileMenu.tsx#L88)
  - **Description**: `'/settings/api-keys' as FileRouteTypes['to']` repeats the cast pattern used for the adjacent menu items, but adds yet another hand-typed path. If TanStack regenerates the route tree and renames, this `as` cast silently swallows the mismatch.
  - **Impact**: Minor — the cast is already the pattern in this file, so this matches existing style.
  - **Suggestion**: Out of scope for this PR, but a future cleanup could route all these through `pathCreator` helpers (like `pathCreator.billing()` two lines below) for type safety.

## 5. Questions

- [ ] What happens on the backend when `workspaceID: null` is passed to `account.apiKeys` — is it "keys across all envs in the account" or "keys with no env"? Worth confirming the semantics match the current UI assumption.
- [ ] The PR description mentions "legacy keys… render as `sk-inn-api••••`; new keys render as `sk-inn-api••••<preview>`". Is the preview suffix server-side-only? If so, the UI is a pure pass-through and no client slicing is needed (the code correctly renders `maskedKey` verbatim, which is good).
- [ ] The create flow requires an `Environment` selection, but the PR description notes the spec required this while Figma didn't have it. Do we have a plan for a sensible default (e.g., pre-select production) to reduce friction, especially for the common case?
- [ ] Are API keys per-account, per-workspace, or scoped by role? The delete modal warns "Any application using this key will immediately lose access" but the UI has no "show affected apps" hint. Not a blocker, but worth considering for a future iteration.
- [ ] Do we want analytics/telemetry (create/reveal/rename/delete events) for this surface to track adoption?

## 6. Positive Callouts ✨

- [x] **Modal state lifted to the route** ([index.tsx:23-28](ui/apps/dashboard/src/routes/_authed/settings/api-keys/index.tsx#L23-L28)) with a comment explaining *why*: it survives the empty→populated transition that unmounts `EmptyState`. This is exactly the right call and the comment captures the *why* rather than the *what*.
- [x] **Stable `queryContext` at module scope** ([useAPIKeys.ts:27](ui/apps/dashboard/src/components/APIKeys/useAPIKeys.ts#L27)) to avoid an urql render loop — the PR description documents the failure mode clearly.
- [x] **Branch-parent filtering** ([CreateAPIKeyModal.tsx:47-56](ui/apps/dashboard/src/components/APIKeys/CreateAPIKeyModal.tsx#L47-L56)) comes with a comment that explains the product reason, not the mechanic.
- [x] **Plaintext never in urql cache**: read once from mutation response, stored only in local React state, dropped on close. No "copy as cURL" surface. Matches the security notes in the spec.
- [x] **`additionalTypenames: ['APIKey']`** is applied consistently on all three mutations and the query. Correct urql pattern for document-cache invalidation when the list may start empty.
- [x] **Reuse of shared primitives** — `Modal`, `AlertModal`, `Select`, `Table`, `Alert`, `CopyButton` — keeps styling and a11y consistent with the rest of the dashboard.
- [x] **Commit history is clean and well-scoped**: the 3 commits (feat → refactor to shared primitives → hide branch parents) read well in isolation.
