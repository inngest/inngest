/**
 * What the canvas is claiming, and how sure it is.
 *
 * Three line styles carry meaning, and two of them say "this may not be a
 * dependency" — which is unreadable without being told. Collapsed by default so
 * it costs nothing once you know it.
 */
import { useState } from 'react';
import { RiArrowDownSLine, RiArrowRightSLine } from '@remixicon/react';

const LINE = 'h-[2px] w-6 shrink-0 rounded-full';

/** The same token the edges are drawn from, so the samples match the canvas. */
const INK = 'rgb(var(--color-foreground-subtle))';

const dashes = (on: number, off: number) => ({
  backgroundImage: `repeating-linear-gradient(to right, ${INK} 0 ${on}px, transparent ${on}px ${
    on + off
  }px)`,
});

const SAMPLES: { label: string; meaning: string; style: React.CSSProperties }[] = [
  {
    label: 'Waited for it',
    meaning: 'Reported by the SDK.',
    style: { backgroundColor: INK },
  },
  {
    label: 'Lost a race',
    meaning: 'Could have unblocked it. Did not.',
    style: dashes(7, 4),
  },
  {
    label: 'Might have',
    meaning: 'Finished in time. Nothing says it did.',
    style: dashes(2, 3),
  },
];

export function CanvasLegend() {
  const [open, setOpen] = useState(false);

  return (
    <div className="border-subtle bg-canvasBase text-subtle overflow-hidden rounded border text-[11px]">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        className="hover:bg-canvasMuted text-subtle flex w-full items-center gap-1 px-1.5 py-1 leading-none"
      >
        {open ? (
          <RiArrowDownSLine className="h-3 w-3 shrink-0" />
        ) : (
          <RiArrowRightSLine className="h-3 w-3 shrink-0" />
        )}
        Legend
      </button>

      {open && (
        <div className="border-subtle max-h-32 max-w-[17rem] overflow-y-auto border-t px-2 py-1.5">
          <table className="border-separate border-spacing-x-1.5 border-spacing-y-1">
            <tbody>
              {SAMPLES.map((sample) => (
                <tr key={sample.label}>
                  <td className="align-middle">
                    <div className={LINE} style={sample.style} />
                  </td>
                  <td className="text-basis whitespace-nowrap align-middle leading-none">
                    {sample.label}
                  </td>
                  <td className="text-muted align-middle leading-snug">{sample.meaning}</td>
                </tr>
              ))}
              <tr>
                <td className="align-middle">
                  <div className="border-subtle bg-canvasBase mx-auto h-2.5 w-2.5 rounded-full border" />
                </td>
                <td className="text-basis whitespace-nowrap align-middle leading-none">Junction</td>
                <td className="text-muted align-middle leading-snug">
                  Where the run splits or rejoins.
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
