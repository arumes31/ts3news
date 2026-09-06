(function(){
 'use strict';
 const $=id=>document.getElementById(id);if(!$('abyssTalentTitle'))return;
 let data,cls,sub,revision=-1;let foundations={},specializations={};
 const el=(tag,text)=>{const n=document.createElement(tag);if(text!==undefined)n.textContent=text;return n;};
 function idsFor(tree){return tree.subclass?(specializations[tree.id]||[]):(foundations[tree.id]||[]);}
 function inspect(text){$('abyssTalentInspect').textContent=text;}
 function nodeFor(tree,id){return tree.nodes.find(n=>n.id===id);}
 function budget(tree){const p=data.class_progress[tree.class];return tree.subclass?Math.max(0,p.points-5):Math.min(5,p.points);}
 function canAdd(tree,node,ids){
  if(data.locked)return 'Talents are locked during this run. Bank or end it first.';
  if(tree.subclass&&(foundations[cls.id]||[]).length!==5)return 'Choose all five foundation tiers first.';
  const chosen=ids.map(id=>nodeFor(tree,id)).filter(Boolean),same=chosen.filter(n=>n.tier===node.tier);
  const limit=tree.subclass&&node.tier<5?2:1;
  if(same.length>=limit&&limit===2)return 'Choose at most two talents in this tier. Remove one to change the choice.';
  const replacing=limit===1?same.length:0;
  if(ids.length-replacing>=budget(tree))return 'Earn the next class talent point from Abyss monster kills.';
  if(node.tier>0&&!chosen.some(n=>n.tier===node.tier-1))return 'Choose a connected talent in the previous tier first.';
  if(node.tier===5&&chosen.filter(n=>n.tier<5).length<9)return 'Choose nine subclass talents before selecting a final talent.';
  return '';
 }
 function prune(tree,ids){
  let kept=ids.slice();for(let tier=1;tier<6;tier++)if(!kept.some(id=>nodeFor(tree,id).tier===tier-1))kept=kept.filter(id=>nodeFor(tree,id).tier<tier);
  if(tree.subclass&&kept.filter(id=>nodeFor(tree,id).tier<5).length<9)kept=kept.filter(id=>nodeFor(tree,id).tier<5);
  return kept;
 }
 function toggle(tree,node){
  let chosen=idsFor(tree).slice();inspect(node.name+'. '+node.description);
  if(data.locked){inspect('Run active: you can inspect talents, but changes unlock after banking.');return;}
  if(chosen.includes(node.id))chosen=prune(tree,chosen.filter(id=>id!==node.id));
  else {const reason=canAdd(tree,node,chosen);if(reason){inspect(node.description+' '+reason);return;}
   if(!tree.subclass||node.tier===5)chosen=chosen.filter(id=>nodeFor(tree,id).tier!==node.tier);
   chosen.push(node.id);
  }
  if(tree.subclass)specializations[tree.id]=chosen;else foundations[tree.id]=chosen;
  draw();document.querySelector('[data-talent="'+node.id+'"]').focus({preventScroll:true});inspect(node.name+'. '+node.description+' Unsaved path changes.');
 }
 function drawTree(tree,target){
  target.replaceChildren();const title=el('h4',tree.subclass?sub.name+' talent tree':cls.name+' foundation');target.append(title);
  const description=el('p',tree.subclass?'Choose up to two talents per tier. After nine choices, select one final talent. Ten points maximum.':'Choose one talent in each tier. Five points unlock the subclass choice. Competing choices stay unavailable in this build.');target.append(description);
  const chosen=idsFor(tree);for(let tier=0;tier<(tree.subclass?6:5);tier++){
   const row=el('div');row.className='ab-talent-tier';row.setAttribute('role','group');row.setAttribute('aria-label',(tier===5?'Final talents':'Tier '+(tier+1)));
   tree.nodes.filter(n=>n.tier===tier).forEach(node=>{
    const button=el('button');button.type='button';button.className='ab-talent-node';button.dataset.talent=node.id;button.setAttribute('aria-pressed',String(chosen.includes(node.id)));
    const reason=chosen.includes(node.id)?'':canAdd(tree,node,chosen);button.setAttribute('aria-disabled',String(!!reason||data.locked));
    const image=el('img');image.src=node.art;image.alt='';image.width=64;image.height=64;image.loading='lazy';
    button.append(image,el('strong',node.name.split(' · ').pop()),el('small',node.description));
    button.setAttribute('aria-label',node.name+'. '+node.description+(reason?' '+reason:''));button.addEventListener('click',()=>toggle(tree,node));row.append(button);
   });target.append(row);
  }
 }
 function draw(){
  if(!data||!data.talent_catalog)return;const p=data.class_progress[cls.id],foundation=foundations[cls.id]||[];
  const isSub=sub.id!==cls.id;const selected=isSub?(specializations[sub.id]||[]):[];
  $('abyssTalentTitle').textContent=cls.name+' progression';$('abyssTalentBudget').textContent=p.points+' / 15 points earned · '+foundation.length+'/5 foundation'+(isSub?' · '+selected.length+'/10 subclass':'');
  const xp=(p.xp/1000).toLocaleString(undefined,{maximumFractionDigits:3});
  $('abyssTalentProgress').textContent=xp+' class XP · '+p.clears+' combat floors cleared with this class.'+(p.next_xp?' Next point at '+(p.next_xp/1000).toLocaleString()+' class XP.':' All fifteen points earned. Your build still cannot learn every talent.');
  $('abyssTalentXP').max=p.next_xp||Math.max(1,p.xp);$('abyssTalentXP').value=p.xp;
  $('abyssTalentPacing').textContent=(p.legacy_credit?'Existing subclass preserved: the first five points were credited once. ':'')+'First five points: 1, 5, 15, 35 and 75 class XP. Later points take 500–5,000 more qualifying combat floors. XP is permanent; a normal complete combat floor grants 1 XP, bosses up to 10% extra.'+(p.points>=5&&p.best_depth>=100?' Floors below '+Math.floor(p.best_depth/4)+' now grant 25% class XP.':'');
  drawTree(data.talent_catalog[cls.id],$('abyssFoundationTree'));
  const gate=$('abyssSubclassGate');gate.replaceChildren(el('h4',foundation.length===5?'Choose your subclass':'Subclass choice · '+foundation.length+' of 5 foundation tiers chosen'));
  const choices=el('div');choices.className='ab-talent-subclasses';cls.subclasses.forEach(choice=>{const b=el('button',choice.name);b.type='button';b.dataset.subclassChoice=choice.id;b.setAttribute('aria-pressed',String(sub.id===choice.id));b.addEventListener('click',()=>{const select=$('abyssSubclass');select.value=choice.id;select.dispatchEvent(new Event('change'));});choices.append(b);});gate.append(choices,el('p','Inspect either path. Activate one after saving five foundation talents; the subclass choice costs no extra point.'));
  $('abyssSubclassTree').hidden=!isSub;if(isSub)drawTree(data.talent_catalog[sub.id],$('abyssSubclassTree'));
  $('abyssTalentSave').disabled=data.locked;$('abyssTalentReset').disabled=data.locked;
 }
 window.AbyssClassTalents={render:function(next,chosenClass,chosenSub){
  data=next;cls=chosenClass;sub=chosenSub;
  if(revision!==next.state.revision||window.__abyssTalentData!==next){
   foundations={};specializations={};for(const c of next.catalog)foundations[c.id]=[...(next.class_progress[c.id].foundation||[])];
   for(const [id,p]of Object.entries(next.state.profiles))specializations[id]=[...(p.talents||[])];revision=next.state.revision;window.__abyssTalentData=next;
  }draw();
 }};
 $('abyssTalentReset').addEventListener('click',()=>{if(data.locked)return;if(sub.id===cls.id)foundations[cls.id]=[];else specializations[sub.id]=[];draw();inspect('Path reset in this preview. Save to apply it; earned XP is preserved.');});
 $('abyssTalentSave').addEventListener('click',async()=>{
  if(data.locked)return;const foundation=foundations[cls.id]||[];const selected=sub.id!==cls.id&&foundation.length===5?sub.id:'';
  const body={class:cls.id,selected,foundation};
  if(selected){const profile=data.state.profiles[selected]||{};body.profile={skills:profile.skills??data.skills.filter(s=>s.source!=='class_signature'&&data.learned.some(l=>l.id===s.id)).map(s=>s.id),pins:profile.pins||[],talents:specializations[selected]||[]};}
  $('abyssTalentSave').disabled=true;await window.AbyssClassWorkbench.saveTalents(body);$('abyssTalentSave').disabled=data.locked;
 });
})();
