// Source inventory only: never executes the application or assumes a string is translatable.
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto'),acorn=require('acorn');
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
function inventory(sources){
 const grouped=new Map(),files=[];
 for(const [file,source] of [...sources].sort((a,b)=>a[0].localeCompare(b[0],'en'))){
  files.push({file,sha256:hash(source)});
  for(const item of extractStrings(file,source)){
   const id=hash(item.kind+'\0'+item.text),existing=grouped.get(id)||{id,kind:item.kind,text:item.text,translation:null,review:'unreviewed',occurrences:[]};
   existing.occurrences.push({file:item.file,line:item.line,column:item.column,context:item.context,parameters:item.parameters});grouped.set(id,existing);
  }
 }
 return {schema:1,sourceLanguage:'en',scope:'rift*.js static string candidates; excludes HTML and Go',files,entries:[...grouped.values()].sort((a,b)=>a.id.localeCompare(b.id,'en'))};
}
if(require.main===module){
 try{
  const args=process.argv.slice(2);if(args.length&&!(args.length===2&&args[0]==='--out'))throw new Error('Usage: node scripts/brawl-string-inventory.cjs [--out FILE]');
  const root=path.resolve(__dirname,'..'),dir=path.join(root,'internal/bot/webassets');
  const sources=fs.readdirSync(dir).filter(file=>/^rift.*\.js$/.test(file)).map(file=>['internal/bot/webassets/'+file,fs.readFileSync(path.join(dir,file),'utf8')]);
  const text=JSON.stringify(inventory(sources),null,2)+'\n';
  if(args.length)fs.writeFileSync(args[1],text.replace(/\n/g,'\r\n'),'utf8');else process.stdout.write(text);
 }catch(error){process.stderr.write(error.message+'\n');process.exitCode=1;}
}
module.exports={extractStrings,inventory};
