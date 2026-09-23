const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,ribbon,wire,ROW,TOP,LBL,PLOT,W,setLiteral} from './micro.mjs';
setLiteral(true);
const E=86;

/**
 * Figures plus their margin notes. A note names a target inside the plot —
 * either a point (`at`) or a range (`span`), in plot percent, plus the row —
 * so the leader can be drawn at the exact thing the note is about.
 */
const O={
  bars:{
    svg:fig([
      {n:'a',at:[['queued',0],['started',10],['ok',86]],note:'40ms'},
      {n:'b',at:[['queued',0],['started',8],['retry',34],['queued',46],['started',50],['ok',86]],note:'recovered'},
      {n:'wait',at:[['queued',0],['started',6],['timeout',86]],kind:'wait',note:'nothing matched'},
      {n:'live',at:[['queued',0],['started',8]],end:86,note:'undecided'},
    ],'','bar colours follow what the code did'),
    notes:[
      {side:'left', span:[0,10], row:0, text:'hatched — not your compute'},

      {side:'right',at:50,       row:2, text:'returns nothing → grey'},
      {side:'right',at:50,       row:3, text:'still going → amber'},
      {side:'left', at:20,       row:1, text:'rejects → red'},
    ],
  },
  events:{
    svg:fig([
      {n:'req + a',at:[['queued',0],['started',4.9,'disc'],['ok',14],['planned',14],['started',18],['ok',86]],reported:1,note:'planned it'},
      {n:'b',at:[['planned',14],['started',18],['ok',80]],note:'planned'},
      {n:'c',at:[['queued',60],['started',66],['ok',86]],note:'discovered'},
    ],'','the event marks'),
    notes:[
      {side:'left', at:0,    row:0, text:'queued — a discovery request'},
      {side:'left', at:14,   row:1, text:'blue — a step already planned'},
      {side:'right',at:E,    row:0, text:'filled — done'},
      {side:'right',at:E-26, row:2, text:'grey — discovered its own work'},
    ],
  },
  structure:{
    svg:fig([
      {n:'req + a',at:[['queued',0],['started',4.2,'disc'],['ok',12],['planned',12],['started',15],['ok',37]],reported:1},
      {n:'b',at:[['planned',12],['started',15],['ok',45]]},
      {n:'c',at:[['planned',12],['started',15],['ok',41]]},
      {n:'d',at:[['queued',45],['started',55],['ok',86]]},
    ],
      '',
      'ribbon and cables'),
    notes:[
      {side:'left', span:[0,12], row:0, text:'discovery — it planned several'},
      {side:'left', at:12,       row:2, text:'ribbon — the steps it planned'},
      {side:'right',at:45,       row:3, text:'cables — what caused this one'},
    ],
  },
};
O.geom={ROW,TOP,LBL,PLOT,W};
fs.writeFileSync(HERE+'annotated.json',JSON.stringify(O));
console.log('ok',Object.keys(O).filter(k=>k!=='geom').join(' '));
