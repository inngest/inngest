const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,ribbon,wire,ROW,TOP,LBL,PLOT,W} from './micro.mjs';
const E=86;

/**
 * Figures plus their margin notes. A note names a target inside the plot —
 * either a point (`at`) or a range (`span`), in plot percent, plus the row —
 * so the leader can be drawn at the exact thing the note is about.
 */
const O={
  bars:{
    svg:fig([
      {n:'a',segs:[['idle',0,10],['good',10,E-10]],note:'40ms'},
      {n:'b',segs:[['idle',0,8],['bad',8,26],['backoff',34,12],['idle',46,4],['good',50,E-50]],note:'recovered'},
      {n:'wait',segs:[['idle',0,6],['waitout',6,E-6]],note:'nothing matched'},
      {n:'live',segs:[['idle',0,8],['running',8,E-8]],note:'undecided'},
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
      {n:'req + a',segs:[['disc',0,14],['idle',14,4],['good',18,E-18]],note:'planned it'},
      {n:'b',segs:[['idle',14,4],['good',18,E-24]],note:'planned'},
      {n:'c',segs:[['idle',E-26,6],['good',E-20,20]],note:'discovered'},
    ],'','the event marks',{rib:{x:14,rows:[0,1]}}),
    notes:[
      {side:'left', at:0,    row:0, text:'queued — a discovery request'},
      {side:'left', at:14,   row:1, text:'blue — a step already planned'},
      {side:'right',at:E,    row:0, text:'filled — done'},
      {side:'right',at:E-26, row:2, text:'grey — discovered its own work'},
    ],
  },
  structure:{
    svg:fig([
      {n:'req + a',segs:[['disc',0,12],['idle',12,3],['good',15,22]]},
      {n:'b',segs:[['idle',12,3],['good',15,30]]},
      {n:'c',segs:[['idle',12,3],['good',15,26]]},
      {n:'d',segs:[['idle',45,10],['good',55,E-55]]},
    ],
      wire(px(37),cy(0),px(45),cy(3),{i:0})+wire(px(45),cy(1),px(45),cy(3),{i:2})+wire(px(41),cy(2),px(45),cy(3),{i:1}),
      'ribbon and cables',{rib:{x:12,rows:[0,1,2]}}),
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
