const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,tag,C,W} from './micro.mjs';
const O={};

// Same shape either way — the point of the figure.
O.same=fig([
  {n:'http · a',segs:[['idle',0,22.0],['good',22,40]],note:'fan-out owner'},
  {n:'connect · a',segs:[['idle',0,22.0],['good',22,40]],note:'identical'},
],'','identical trace shape');

// What is missing is inside the row, on expand.
O.expand=fig([
  {n:'http · a',segs:[['idle',0,22],['good',22,40]],note:'40ms'},
  {n:'  dns',segs:[['idle',2,3]],dots:[],dim:.7,note:'3ms'},
  {n:'  tcp',segs:[['idle',5,4]],dots:[],dim:.7,note:'4ms'},
  {n:'  tls',segs:[['idle',9,7]],dots:[],dim:.7,note:'7ms'},
  {n:'  ttfb',segs:[['idle',16,6]],dots:[],dim:.7,note:'6ms'},
  {n:'connect · a',segs:[['idle',0,22],['good',22,40]],note:'40ms'},
],tag(px(2),cy(5)+13,'no network breakdown exists, do not offer an empty one',C.mut),'http timing absent under connect');

// The polling artifact lands inside the green.
O.poll=fig([
  {n:'connect · a',segs:[['idle',0,10],['good',10,74]],note:'up to 5s of this is not your code'},
],`<rect x="${px(46)}" y="${cy(0)-4.5}" width="${px(84)-px(46)}" height="9" fill="${C.bad}" opacity=".16"/>`+
  tag(px(47),cy(0)+13,'proxy poll: 2s + rand(0…3s) if the push is missed',C.bad),'polling inflates execution');

fs.writeFileSync(HERE+'connect.json',JSON.stringify(O));
console.log('ok',Object.keys(O).join(' '));
