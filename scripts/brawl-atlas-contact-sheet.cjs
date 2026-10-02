// Build a local review page; original image bytes are never changed.
'use strict';
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto');
const {pathToFileURL}=require('node:url');
const root=path.resolve(__dirname,'../internal/bot/webassets');
const shared=['abyss_combat_roles_v2.png','abyss_combat_creatures_v2.png','abyss_combat_bestiary_v2.png','abyss_combat_bosses_v2.png','abyss_player_classes_v1.png','abyss_subclasses_martial_v1.png','abyss_subclasses_mystic_v1.png'];
const escape=value=>String(value).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
function inventory(){
 return [...new Set([...fs.readdirSync(root).filter(name=>/^rift.*\.png$/.test(name)),...shared])].sort().map(name=>{
  const file=path.join(root,name),data=fs.readFileSync(file);
  if(data.subarray(0,8).toString('hex')!=='89504e470d0a1a0a')throw Error('Not a PNG: '+name);
  return {name,file,width:data.readUInt32BE(16),height:data.readUInt32BE(20),bytes:data.length,sha256:crypto.createHash('sha256').update(data).digest('hex')};
 });
}
function render(rows){
 const cards=rows.map(row=>`<article data-name="${escape(row.name)}"><h2>${escape(row.name)}</h2><a href="${escape(pathToFileURL(row.file).href)}" target="_blank" rel="noopener"><img src="${escape(pathToFileURL(row.file).href)}" width="${row.width}" height="${row.height}" alt="Full atlas: ${escape(row.name)}"></a><p>${row.width} × ${row.height} px · ${(row.bytes/1048576).toFixed(2)} MiB</p><details><summary>Source fingerprint</summary><code>${row.sha256}</code></details></article>`).join('\n');
 return `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Brawl atlas contact sheet</title><style>
*{box-sizing:border-box}body{margin:0;background:#101716;color:#eee9da;font:16px/1.5 system-ui,sans-serif}header{padding:28px 32px;border-bottom:1px solid #425148}h1{margin:0;font:32px Georgia,serif}header p{max-width:850px;color:#bfc9bd}label{display:inline-flex;gap:10px;align-items:center;margin:8px 20px 0 0}input,select{font:inherit;padding:9px;background:#1d2c25;color:inherit;border:1px solid #809283;max-width:100%}main{padding:24px;display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,330px),1fr));gap:20px}article{min-width:0;border:1px solid #425148;padding:16px;background:#18231e}h2{font:15px ui-monospace,monospace;overflow-wrap:anywhere}article a{display:flex;align-items:center;justify-content:center;height:290px;background:repeating-conic-gradient(#aaa 0 25%,#ddd 0 50%) 0/20px 20px}img{width:100%;height:100%;object-fit:contain;image-rendering:pixelated}article p{font-size:13px}code{font-size:11px;overflow-wrap:anywhere}summary{cursor:pointer;font-size:13px}[hidden]{display:none!important}body[data-bg=dark] article a{background:#080d0b}body[data-bg=light] article a{background:#fff}a:focus-visible,input:focus-visible,select:focus-visible,summary:focus-visible{outline:3px solid #ffd782;outline-offset:3px}
</style><header><h1>Brawl atlas contact sheet</h1><p>Full-sheet review of Rift artwork and shared Abyss combat/class atlases. Click an image to inspect the unchanged original at full resolution. Thumbnails preserve aspect ratio; this sheet does not certify sprite crops or animation alignment.</p><label>Find atlas <input type="search" id="filter"></label><label>Transparency background <select id="background"><option value="checker">Checkerboard</option><option value="dark">Dark</option><option value="light">Light</option></select></label><p id="count" role="status">${rows.length} atlases</p></header><main>${cards}</main><script>
const cards=[...document.querySelectorAll('article')];document.querySelector('#filter').addEventListener('input',event=>{const query=event.target.value.toLowerCase().trim();let visible=0;for(const card of cards){card.hidden=!card.dataset.name.toLowerCase().includes(query);if(!card.hidden)visible++;}document.querySelector('#count').textContent=visible+' of '+cards.length+' atlases';});document.querySelector('#background').addEventListener('change',event=>document.body.dataset.bg=event.target.value);
</script></html>`;
}
if(require.main===module){const output=path.resolve(process.argv[2]||'.tmp/brawl-atlas-contact-sheet.html');const rows=inventory();fs.writeFileSync(output,render(rows),'utf8');console.log(JSON.stringify({output,atlases:rows.length,totalBytes:rows.reduce((sum,row)=>sum+row.bytes,0)}));}
module.exports={inventory,render};
