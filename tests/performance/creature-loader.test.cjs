const test=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path'),vm=require('node:vm');
const root=path.resolve(__dirname,'../../internal/bot/webassets');
function harness(decode,contextAvailable=true){
 const draws=[],canvases=[],scope={window:{},document:{createElement(tag){assert.equal(tag,'canvas');const canvas={width:0,height:0,getContext(){return contextAvailable?{clearRect(...rect){draws.push({clear:rect});},drawImage(image,...position){draws.push({image,position});}}:null;}};canvases.push(canvas);return canvas;}}};
 vm.runInNewContext(fs.readFileSync(path.join(root,'rift_creature_sections.js'),'utf8'),scope);
 vm.runInNewContext(fs.readFileSync(path.join(root,'rift_creature_loader.js'),'utf8'),scope);
 const manifest=scope.window.RiftCreatureSections;
 const loader=scope.window.RiftCreatureLoader.create({manifest,decode});
 const frame=rig=>({rig,asset:manifest.rigs[rig].sourceAsset,source:{x:0,y:manifest.rigs[rig].rowY/1254,width:1/8,height:manifest.rigs[rig].rowHeight/1254}});
 return {loader,manifest,frame,draws,canvases};
}
const dimensions=(manifest,rig)=>({width:manifest.rigs[rig].width,height:manifest.rigs[rig].height});

test('concurrent preparation shares a decode and never exposes another blank row',async()=>{
 let release,calls=0;const pending=new Promise(resolve=>release=resolve);
 const h=harness(()=>{calls++;return pending;});
 const frame=h.frame('ranger'),first=h.loader.prepare([frame]),second=h.loader.prepare([frame]);
 await Promise.resolve();assert.equal(calls,1);assert.equal(h.loader.image(frame),null);
 release(dimensions(h.manifest,'ranger'));await Promise.all([first,second]);
 assert.ok(h.loader.image(frame));assert.equal(h.loader.image(h.frame('wizard')),null);
 assert.equal(h.loader.stats().surfaces,1);assert.equal(h.loader.stats().preparedRows,1);assert.equal(h.loader.stats().pendingRows,0);
 assert.equal(h.draws[0].clear[1],h.manifest.rigs.ranger.top);assert.equal(h.draws[1].position[1],h.manifest.rigs.ranger.top);
});

test('dimension failure releases the bitmap and retries a fresh URL',async()=>{
 const urls=[];let closed=0;let h;
 h=harness(src=>{urls.push(src);return Promise.resolve({...dimensions(h.manifest,'dragon'),height:urls.length===1?1:h.manifest.rigs.dragon.height,close(){closed++;}});});
 await assert.rejects(h.loader.prepare([h.frame('dragon')]),/creature artwork/i);
 assert.equal(h.loader.stats().surfaces,0);assert.equal(h.loader.stats().pendingRows,0);
 await Promise.all([h.loader.prepare([h.frame('dragon')]),h.loader.prepare([h.frame('dragon')])]);
 assert.equal(urls.length,2);assert.match(urls[1],/[?&]retry=1$/);assert.equal(closed,2);
});

test('all rows retain only four original-sized surfaces',async()=>{
 let h;h=harness(src=>{const rig=Object.keys(h.manifest.rigs).find(key=>h.manifest.rigs[key].url===src);return Promise.resolve(dimensions(h.manifest,rig));});
 const frames=Object.keys(h.manifest.rigs).map(h.frame);await h.loader.prepare(frames);
 assert.equal(h.loader.stats().surfaces,4);assert.equal(h.loader.stats().preparedRows,32);assert.equal(h.loader.stats().surfaceBytes,4*1254*1254*4);assert.equal(h.loader.stats().pendingRows,0);
 h.loader.dispose();assert.equal(h.loader.stats().surfaceBytes,0);assert.equal(h.loader.image(frames[0]),null);assert.ok(h.canvases.every(c=>c.width===0&&c.height===0));
});

test('disposal during decode prevents publication and closes the late bitmap',async()=>{
 let release,closed=0;const h=harness(()=>new Promise(resolve=>release=resolve));
 const pending=h.loader.prepare([h.frame('rat')]);await Promise.resolve();h.loader.dispose();
 release({...dimensions(h.manifest,'rat'),close(){closed++;}});await assert.rejects(pending,/closed/i);
 assert.equal(closed,1);assert.equal(h.loader.stats().surfaces,0);assert.equal(h.loader.stats().pendingRows,0);
 await assert.rejects(h.loader.prepare([h.frame('rat')]),/closed/i);
});

test('missing legacy frames need no shared sheet; stale known frames fail visibly',async()=>{
 const h=harness(()=>{throw Error('Unexpected decode');});
 await h.loader.prepare([null,{rig:'retired',asset:'/static/old.png',source:null}]);assert.equal(h.loader.stats().surfaces,0);
 await assert.rejects(h.loader.prepare([{rig:'future-rig',asset:'/static/unknown.png',source:{x:0,y:0,width:1,height:1}}]),/manifest/i);
});

test('unavailable canvas contexts leave no retained surface',async()=>{
 let closed=0,h;h=harness(()=>Promise.resolve({...dimensions(h.manifest,'rat'),close(){closed++;}}),false);
 await assert.rejects(h.loader.prepare([h.frame('rat')]),/creature artwork/i);
 assert.equal(closed,1);assert.equal(h.loader.stats().surfaces,0);assert.equal(h.canvases[0].width,0);
});

test('invalid frame lists fail before decoding any rows, including inherited property names',async()=>{
 let calls=0,h;h=harness(()=>{calls++;return Promise.resolve(dimensions(h.manifest,'rat'));});
 for(const rig of ['future-rig','toString','__proto__']){
  await assert.rejects(h.loader.prepare([h.frame('rat'),{rig,source:{x:0,y:0,width:1,height:1}}]),/manifest/i);
  assert.equal(calls,0);assert.equal(h.loader.stats().pendingRows,0);
 }
});
