'use strict';
const fs=require('node:fs');
const path=require('node:path');
const types={'.js':['application/javascript','text/javascript'],'.css':['text/css'],'.png':['image/png'],'.webp':['image/webp'],'.svg':['image/svg+xml'],'.ttf':['font/ttf','application/x-font-ttf']};
function assetManifest(root){
 const pending=['rift.html','partials.html'],seen=new Set(),assets=new Set();
 // These filenames are assembled at runtime in the shared combat-art selector.
 for(const file of ['abyss_subclasses_martial_v1.png','abyss_subclasses_mystic_v1.png','abyss_player_classes_v1.png']){assets.add('/static/'+file);pending.push(file);}
 while(pending.length){
  const file=pending.pop();if(seen.has(file))continue;seen.add(file);
  if(file.split('/').some(part=>!part||part==='.'||part==='..')||file.includes('\\'))throw new Error('Invalid asset path: '+file);
  const location=path.join(root,file),stat=fs.statSync(location);
  if(!stat.isFile()||stat.size===0)throw new Error('Empty or non-file asset: '+file);
  if(!/\.(html|js|css)$/.test(file))continue;
  const source=fs.readFileSync(location,'utf8');
  for(const match of source.matchAll(/\/static\/([A-Za-z0-9][A-Za-z0-9_./-]*\.(?:js|css|png|webp|svg|ttf))\b/g)){
   assets.add('/static/'+match[1]);pending.push(match[1]);
  }
 }
 return [...assets].sort();
}
async function checkAssets(base,assets,fetcher=fetch){
 const results=new Array(assets.length);let cursor=0;
 async function worker(){
  while(cursor<assets.length){
   const index=cursor++,asset=assets[index],row={asset,ok:false};
   try{
    const response=await fetcher(new URL(asset,base),{method:'HEAD',redirect:'manual',headers:{'Cache-Control':'no-cache'},signal:AbortSignal.timeout(5000)});
    row.status=response.status;row.contentType=(response.headers.get('content-type')||'').split(';')[0].trim().toLowerCase();
    if(response.status!==200)row.error='Expected HTTP 200';
    else if(!types[path.extname(asset)]?.includes(row.contentType))row.error='Unexpected content type';
    else if(response.headers.get('content-length')==='0')row.error='Empty asset';
    else row.ok=true;
   }catch(error){row.error=error.message;}
   results[index]=row;
  }
 }
 await Promise.all(Array.from({length:Math.min(4,assets.length)},worker));return results;
}
async function main(){
 try{
  if(process.argv.length!==3)throw new Error('Usage: node scripts/check-brawl-assets.cjs http://127.0.0.1:18096');
  const base=new URL(process.argv[2]);
  if(!['http:','https:'].includes(base.protocol)||base.username||base.password||base.search||base.hash||base.pathname!=='/')throw new Error('Supply an HTTP(S) origin without credentials, path, query or fragment');
  const assets=assetManifest(path.resolve(__dirname,'../internal/bot/webassets'));
  const results=await checkAssets(base,assets),failed=results.filter(row=>!row.ok).length;
  process.stdout.write(JSON.stringify({checked:results.length,failed,results},null,2)+'\n');
  process.stderr.write(`Brawl assets: ${results.length-failed}/${results.length} available\n`);
  process.exitCode=failed?1:0;
 }catch(error){process.stderr.write(error.message+'\n');process.exitCode=2;}
}
module.exports={assetManifest,checkAssets};
if(require.main===module)main();
