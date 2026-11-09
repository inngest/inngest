import { act, cleanup, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import * as v4 from '../RunDetailsV4/runDetailsUtils';
import type { StepInfoType } from '../RunDetailsV4/runDetailsUtils';
import * as shared from './utils';

afterEach(cleanup);

describe.each([
  ['V4', v4],
  ['shared', shared],
] as const)(
  '%s run-scoped subscription state',
  (_name, { useStepSelection, useDynamicRunData }) => {
    it('clears the selected step when switching runs', () => {
      const rendered: (string | undefined)[] = [];
      const { result, rerender } = renderHook(
        ({ runID }) => {
          const state = useStepSelection({ runID });
          rendered.push(state.selectedStep?.runID);
          return state;
        },
        { initialProps: { runID: 'run-a' } }
      );
      const step = { runID: 'run-a', trace: { spanID: 'step-a' } } as StepInfoType &
        shared.StepInfoType;
      act(() => result.current.selectStep(step));
      expect(result.current.selectedStep).toBe(step);
      rerender({ runID: 'run-a' });
      expect(result.current.selectedStep).toBe(step);
      rendered.length = 0;
      rerender({ runID: 'run-b' });
      expect(rendered).not.toContain('run-a');
      expect(result.current.selectedStep).toBeUndefined();
      act(() => result.current.selectStep(step));
      expect(result.current.selectedStep).toBeUndefined();
      const nextStep = { ...step, runID: 'run-b' };
      act(() => result.current.selectStep(nextStep));
      expect(result.current.selectedStep).toBe(nextStep);
      act(() => result.current.selectStep(undefined));
      expect(result.current.selectedStep).toBeUndefined();
    });
    it('keeps unscoped observers listening to any run', () => {
      const { result } = renderHook(() => useStepSelection({}));
      const step = { runID: 'run-a', trace: { spanID: 'step-a' } } as StepInfoType &
        shared.StepInfoType;
      act(() => result.current.selectStep(step));
      expect(result.current.selectedStep).toBe(step);
      const nextStep = { ...step, runID: 'run-b' };
      act(() => result.current.selectStep(nextStep));
      expect(result.current.selectedStep).toBe(nextStep);
    });
    it('clears dynamic status when switching runs', () => {
      const { result, rerender } = renderHook(({ runID }) => useDynamicRunData({ runID }), {
        initialProps: { runID: 'run-a' },
      });
      act(() => result.current.updateDynamicRunData({ runID: 'run-a', status: 'COMPLETED' }));
      expect(result.current.dynamicRunData?.status).toBe('COMPLETED');
      rerender({ runID: 'run-b' });
      expect(result.current.dynamicRunData).toBeUndefined();
      act(() => result.current.updateDynamicRunData({ runID: 'run-a', status: 'FAILED' }));
      expect(result.current.dynamicRunData).toBeUndefined();
      act(() => result.current.updateDynamicRunData({ runID: 'run-b', status: 'RUNNING' }));
      expect(result.current.dynamicRunData?.status).toBe('RUNNING');
      rerender({ runID: 'run-a' });
      expect(result.current.dynamicRunData).toBeUndefined();
    });
  }
);
