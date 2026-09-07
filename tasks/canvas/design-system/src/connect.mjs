const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,tag,C,W} from './micro.mjs';
const O={};

// Same shape either way — the point of the figure.
O.same=fig([
  {n:'http · a',at:[['queued',0],['started',22],['ok',62]],note:'fan-out owner'},
  {n:'connect · a',at:[['queued',0],['started',22],['ok',62]],note:'identical'},
],'','identical trace shape');

// What is missing is inside the row, on expand.
O.expand=fig([
  {n:'http · a',at:[['queued',0],['started',22],['ok',62]],note:'40ms'},
  {n:'  dns',at:[['queued',2]],end:5,dots:[],dim:.7,note:'3ms'},
  {n:'  tcp',at:[['queued',5]],end:9,dots:[],dim:.7,note:'4ms'},
  {n:'  tls',at:[['queued',9]],end:16,dots:[],dim:.7,note:'7ms'},
  {n:'  ttfb',at:[['queued',16]],end:22,dots:[],dim:.7,note:'6ms'},
  {n:'connect · a',at:[['queued',0],['started',22],['ok',62]],note:'40ms'},
],tag(px(2),cy(5)+13,'no network breakdown exists, do not offer an empty one',C.mut),'http timing absent under connect');

// The polling artifact lands inside the green.
O.poll=fig([
  {n:'connect · a',at:[['queued',0],['started',10],['ok',84]],note:'up to 5s of this is not your code'},
],`<rect x="${px(46)}" y="${cy(0)-4.5}" width="${px(84)-px(46)}" height="9" fill="${C.bad}" opacity=".16"/>`+
  tag(px(47),cy(0)+13,'proxy poll: 2s + rand(0…3s) if the push is missed',C.bad),'polling inflates execution');

fs.writeFileSync(HERE+'connect.json',JSON.stringify(O));
console.log('ok',Object.keys(O).join(' '));
