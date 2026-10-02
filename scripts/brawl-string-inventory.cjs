// Source inventory only: never executes the application or assumes a string is translatable.
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto'),acorn=require('acorn'),html=require('parse5');
const hash=text=>crypto.createHash('sha256').update(text).digest('hex');
function extractStrings(file,source){
 let tree;try{tree=acorn.parse(source,{ecmaVersion:'latest',locations:true});}catch(error){throw new Error(file+': '+error.message);}
 const entries=[];
 function visit(node,parent){
  if(!node||typeof node!=='object')return;
  let text,kind,parameters=[];
  if(node.type==='Literal'&&typeof node.value==='string'){text=node.value;kind='literal';}
  if(node.type==='TemplateLiteral'){
   kind='template';text=node.quasis.map((part,i)=>(part.value.cooked??part.value.raw)+(i<node.expressions.length?'{'+(i+1)+'}':'')).join('');
   parameters=node.expressions.map(expression=>source.slice(expression.start,expression.end));
  }
  if(text&&text.trim())entries.push({file,line:node.loc.start.line,column:node.loc.start.column+1,kind,text,parameters,context:parent?.type||'root'});
  for(const value of Object.values(node)){
   if(Array.isArray(value)){for(const child of value)if(child?.type)visit(child,node);}
   else if(value?.type)visit(value,node);
  }
 }
 visit(tree,null);return entries;
}
function extractHTML(file,source){
 const templates=[],entries=[];
 // Preserve offsets while hiding template quotes from the HTML parser.
 const masked=source.replace(/{{[\s\S]*?}}/g,raw=>{
  const marker=String.fromCharCode(0xe000+templates.length);
  if(source.includes(marker))throw new Error(file+': template marker collision');
  const mask=raw.replace(/[^\r\n]/g,marker);templates.push({mask:mask.replace(/\r\n/g,'\n'),expression:raw.slice(2,-2).trim()});return mask;
 });
 function add(value,location,context){
  if(!location)return;
  const parameters=[];
  for(const template of templates)if(value.includes(template.mask)){
   parameters.push(template.expression);value=value.replaceAll(template.mask,'{'+parameters.length+'}');
  }
  const text=value.trim();if(!text)return;
  entries.push({file,line:location.startLine,column:location.startCol,kind:'html',text,parameters,context});
 }
 function visit(node){
  if(['script','style'].includes(node.tagName))return;
  if(node.nodeName==='#text')add(node.value,node.sourceCodeLocation,'HTMLText');
  for(const attr of node.attrs||[]){
   const buttonValue=attr.name==='value'&&node.tagName==='input'&&node.attrs.some(a=>a.name==='type'&&['button','submit','reset'].includes(a.value));
   if(['title','alt','placeholder','aria-label','aria-description','aria-valuetext'].includes(attr.name)||buttonValue)add(attr.value,node.sourceCodeLocation?.attrs?.[attr.name],'HTMLAttribute:'+attr.name);
  }
  for(const child of node.childNodes||[])visit(child);
  if(node.content)visit(node.content);
 }
 visit(html.parse(masked,{sourceCodeLocationInfo:true}));return entries;
}
function inventory(sources,goSources=[]){
 const grouped=new Map(),files=[];
 const scanned=sources.map(([file,source])=>({file,sha256:hash(source),strings:(file.endsWith('.html')?extractHTML:extractStrings)(file,source)}));
 for(const scannedFile of [...scanned,...goSources].sort((a,b)=>a.file.localeCompare(b.file,'en'))){
  const {file,sha256,strings}=scannedFile;files.push({file,sha256});
  for(const item of strings){
   const id=hash(item.kind+'\0'+item.text),existing=grouped.get(id)||{id,kind:item.kind,text:item.text,translation:null,review:'unreviewed',occurrences:[]};
   existing.occurrences.push({file:item.file,line:item.line,column:item.column,context:item.context,parameters:item.parameters});grouped.set(id,existing);
  }
 }
 return {schema:1,sourceLanguage:'en',scope:'Brawl JS/HTML, shared Abyss JS, rift/content Go and web_rift Go source candidates',files,entries:[...grouped.values()].sort((a,b)=>a.id.localeCompare(b.id,'en'))};
}
if(require.main===module){
 try{
  const args=process.argv.slice(2);if(args.length&&!(args.length===2&&args[0]==='--out'))throw new Error('Usage: node scripts/brawl-string-inventory.cjs [--out FILE]');
  const root=path.resolve(__dirname,'..'),dir=path.join(root,'internal/bot/webassets');
  const sources=fs.readdirSync(dir).filter(file=>/^(?:rift.*\.(?:js|html)|abyss.*\.js)$/.test(file)).map(file=>['internal/bot/webassets/'+file,fs.readFileSync(path.join(dir,file),'utf8')]);
  const extracted=require('node:child_process').spawnSync('go',['run','./cmd/brawl-string-source'],{cwd:root,encoding:'utf8',maxBuffer:64*1024*1024});
  if(extracted.error||extracted.status!==0)throw new Error('Go inventory failed: '+(extracted.error?.message||extracted.stderr));
  const text=JSON.stringify(inventory(sources,JSON.parse(extracted.stdout)),null,2)+'\n';
  if(args.length)fs.writeFileSync(args[1],text.replace(/\n/g,'\r\n'),'utf8');else process.stdout.write(text);
 }catch(error){process.stderr.write(error.message+'\n');process.exitCode=1;}
}
module.exports={extractStrings,extractHTML,inventory};
