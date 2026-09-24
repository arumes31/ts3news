const {test}=require('node:test');
const assert=require('node:assert/strict');
const path=require('node:path');
const {assetManifest,checkAssets}=require('./check-brawl-assets.cjs');
test('manifest includes Brawl modules, terrain and dynamically selected shared class atlases',()=>{
 const files=assetManifest(path.resolve(__dirname,'../internal/bot/webassets'));
 for(const name of ['rift.js','rift_protocol.js','rift_terrain_cover.png','rift_platform_surface.png','abyss_combat_roles_v2.png','abyss_subclasses_martial_v1.png','abyss_subclasses_mystic_v1.png','abyss_player_classes_v1.png'])assert.ok(files.includes('/static/'+name),name);
 assert.equal(new Set(files).size,files.length);
});
test('health rejects missing assets, redirects, HTML fallbacks and empty responses',async()=>{
 const paths=['ok.png','missing.js','redirect.css','fallback.js','empty.png','offline.png'].map(x=>'/static/'+x);
 let active=0,peak=0;
 const fetcher=async(url,options)=>{
  assert.equal(options.method,'HEAD');assert.equal(options.redirect,'manual');assert.equal(options.headers['Cache-Control'],'no-cache');
  active++;peak=Math.max(peak,active);await new Promise(resolve=>setTimeout(resolve,5));active--;
  const name=new URL(url).pathname.split('/').pop();
  if(name==='offline.png')throw new Error('offline');
  return {status:name==='missing.js'?404:name==='redirect.css'?302:200,headers:new Headers({'content-type':name==='fallback.js'?'text/html':'image/png','content-length':name==='empty.png'?'0':'12'})};
 };
 const results=await checkAssets('http://127.0.0.1:18096',paths,fetcher);
 assert.equal(results.length,6);assert.equal(results.filter(x=>x.ok).length,1);assert.equal(results[0].ok,true);assert.ok(peak<=4);
});
test('manifest follows nested references and rejects a missing dependency',()=>{
 const fs=require('node:fs'),os=require('node:os');
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'brawl-assets-'));
 try{
  for(const name of ['partials.html','abyss_subclasses_martial_v1.png','abyss_subclasses_mystic_v1.png','abyss_player_classes_v1.png'])fs.writeFileSync(path.join(root,name),'fixture');
  fs.writeFileSync(path.join(root,'rift.html'),'/static/a.js');
  fs.writeFileSync(path.join(root,'a.js'),"const nested='/static/missing.png';");
  assert.throws(()=>assetManifest(root),/missing\.png/);
  fs.writeFileSync(path.join(root,'missing.png'),'image');
  assert.ok(assetManifest(root).includes('/static/missing.png'));
 }finally{
  for(const file of fs.readdirSync(root))fs.unlinkSync(path.join(root,file));
  fs.rmdirSync(root);
 }
});
