const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {runProfile,px,cy,C,W,LBL,PLOT,dot,EV} from './micro.mjs';
import * as R from './rules.mjs';
const MONO="font-family='JetBrains Mono, ui-monospace, monospace'";

/**
 * The argument for making the Run row a profile of where the elapsed time went.
 *
 * The profile half is drawn by `runProfile` — the same function that draws the
 * Run row on every other figure in the artifact, with the same rank colours.
 * It used to be a second implementation living here: its own slicing of the
 * axis, its own track, its own dots and its own colour rule, which still said
 * "neutral where a slice is mixed" long after that was replaced. A second
 * drawing of the Run row is the exact defect the Run row itself exists to
 * avoid, so there is one of it now.
 *
 * The top half is deliberately NOT ours: it is a picture of the bar as it
 * renders today, which is what the section is arguing against. It cannot come
 * out of these rules because it was not made by them.
 */

// A real run that retried: measured compute intervals as % of elapsed.
const RETRY=[
  {a:0.3,b:1.2,kind:'good'},   {a:49.6,b:50.2,kind:'good'},
  {a:0,b:0.28,kind:'bad'},     {a:49.15,b:49.43,kind:'good'},
  {a:50.75,b:51.03,kind:'bad'},{a:99.61,b:99.89,kind:'good'},
];

const cap=(t,y)=>`<text x="${LBL}" y="${y}" ${MONO} font-size="7" fill="${C.mut}">${t}</text>`;
const out={};

out.idea=`<svg viewBox="0 0 ${W} 106" role="img" aria-label="The run row today compared with a compute profile">`+
  cap('today: the run’s status, full width, repeating the axis',14)+
  `<text x="2" y="${34+3}" ${MONO} font-size="8.5" fill="${C.mut}">Run</text>`+
  `<rect x="${LBL}" y="${34-4}" width="${PLOT}" height="8" rx="1.2" fill="${C.good}"/>`+
  dot(LBL,34,EV.queued)+dot(LBL+PLOT,34,EV.ok)+
  `<text x="${W-4}" y="${37}" ${MONO} font-size="7.5" fill="${C.mut}" text-anchor="end">2.069s</text>`+
  cap('as an execution profile: grey is elapsed, colour is SDK execution',66)+
  `<g transform="translate(0,${86-cy(0)})">`+
    runProfile(0,{to:100, intervals:RETRY.map(v=>({a:v.a,b:v.b,rank:R.runRank(v.kind)})),
                  resolved:EV.ok})+
  `</g>`+
  `<text x="${W-4}" y="${89}" ${MONO} font-size="7.5" fill="${C.mut}" text-anchor="end">19ms compute / 2.069s</text>`+
  `</svg>`;

fs.writeFileSync(HERE+'runbar.json',JSON.stringify(out));
console.log('ok',Object.keys(out).join(' '));
