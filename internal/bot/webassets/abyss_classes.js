(function(){
 'use strict';
 const root=document.getElementById('abyssMyBuild');if(!root)return;
 const $=id=>document.getElementById(id);let data=null,shownClass=null,busy=false;
 function node(tag,text,cls){const el=document.createElement(tag);if(text!==undefined)el.textContent=text;if(cls)el.className=cls;return el;}
 function status(text){$('abyssClassStatus').textContent=text;}
 async function request(body){const response=await fetch('/api/abyss/classes',{method:body?'POST':'GET',credentials:'same-origin',cache:'no-store',headers:{'Content-Type':'application/json'},body:body?JSON.stringify(body):undefined});const result=await response.json();if(!response.ok||!result.ok)throw new Error(result.error||'Could not load your build. Reload to retry.');return result;}
 function subclass(){return shownClass&&($('abyssSubclass').value?shownClass.subclasses.find(s=>s.id===$('abyssSubclass').value):data.foundation_styles[shownClass.id]);}
 function signed(n){return Number(n)>0?'+'+n:String(n||0);}
 function chooseClass(id){shownClass=data.catalog.find(c=>c.id===id)||data.catalog[0];const index=data.catalog.indexOf(shownClass);$('abyssClassPortrait').style.backgroundPosition='0% '+(index*20)+'%';$('abyssClassName').textContent=shownClass.name;$('abyssClassDescription').textContent=shownClass.description;
  $('abyssClassRoster').querySelectorAll('button').forEach(b=>b.setAttribute('aria-pressed',String(b.dataset.class===shownClass.id)));
  const foundation=node('option',shownClass.name+' foundation');foundation.value='';
  $('abyssSubclass').replaceChildren(foundation,...shownClass.subclasses.map(s=>{const option=node('option',s.name);option.value=s.id;return option;}));
  if(shownClass.subclasses.some(s=>s.id===data.state.selected))$('abyssSubclass').value=data.state.selected;
  renderSubclass();
 }
 function renderSubclass(){const sub=subclass();if(!sub)return;
  const isFoundation=sub.id===shownClass.id;
  const subclassIDs=['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'],subIndex=subclassIDs.indexOf(sub.id);
  $('abyssClassPortrait').style.backgroundImage=isFoundation?'url("/static/abyss_player_classes_v1.png")':'url("/static/abyss_subclasses_'+(subIndex<6?'martial':'mystic')+'_v1.png")';
  $('abyssClassPortrait').style.backgroundPosition='0% '+((isFoundation?data.catalog.indexOf(shownClass):subIndex%6)*20)+'%';
  $('abyssClassName').textContent=isFoundation?shownClass.name:shownClass.name+' / '+sub.name;

  $('abyssClassSequence').textContent=sub.sequence;$('abyssClassStats').textContent=sub.stats.join(' → ')+' · '+sub.resource+' 0–3';
  const selected=sub.id===data.state.selected||(isFoundation&&data.state.class===shownClass.id&&!data.state.selected);const progress=data.class_progress[shownClass.id];$('abyssClassChoose').textContent=selected?(isFoundation?'Active class':'Active subclass'):'Use '+sub.name;$('abyssClassChoose').disabled=busy||data.locked||selected||(!isFoundation&&!progress.subclass_unlocked);if(!isFoundation&&!progress.subclass_unlocked)$('abyssClassChoose').textContent='Unlock with five foundation talents';
  $('abyssClassBuffs').textContent=sub.buffs;$('abyssClassGearAdvice').textContent=sub.gear;
  $('abyssClassSignatures').replaceChildren(...data.signatures[sub.id].map((skill,index)=>{const row=node('div',undefined,'ab-class-signature'),title=node('div');title.append(node('h4',skill.name),node('small',skill.role+' · '+skill.mana+' mana · '+skill.cooldown+' round cooldown'+(isFoundation?' · Unlocks at class point '+(index===0?1:3):'')));
   const details=node('div');details.append(node('p',skill.mechanics),node('small',skill.scaling+' scaling · '+skill.base_effect+' base damage'+(skill.heal_percent?' · '+Math.round(skill.heal_percent*100)+'% max-HP healing':'')));row.append(title,details);return row;}));
  if(!selected){$('abyssClassNext').textContent='Choose '+sub.name+' to calculate a reachable upgrade and its point cost for this build.';$('abyssClassGear').replaceChildren();const row=node('tr'),cell=node('td','Activate this subclass to compare your inventory against its priorities.');cell.colSpan=5;row.append(cell);$('abyssClassGear').append(row);}
  else {$('abyssClassNext').textContent=data.next_upgrade.label+'. '+data.next_upgrade.detail;renderGear();}
  renderSkills();
  if(window.AbyssClassTalents)window.AbyssClassTalents.render(data,shownClass,sub);
 }
 function renderSkills(){const sub=subclass(),profile=data.state.profiles[sub.id],fallback=data.skills.filter(s=>s.source!=='class_signature'&&data.learned.some(learned=>learned.id===s.id)).map(s=>s.id);
  const chosen=new Set(profile&&Array.isArray(profile.skills)?profile.skills:fallback),pins=new Set(profile?profile.pins:[]);
  $('abyssClassCapacity').textContent=data.capacity;
  $('abyssClassSkills').replaceChildren(...data.learned.map(skill=>{const row=node('div',undefined,'ab-class-skill-row'),input=node('input');input.type='checkbox';input.value=skill.id;input.checked=chosen.has(skill.id);input.disabled=data.locked;input.setAttribute('aria-label','Equip '+skill.name);input.dataset.equip=skill.id;
   const title=node('div');title.append(node('strong',skill.name),node('small',skill.family+' · rank '+skill.rank));
   const detail=node('small',skill.scaling+' · '+skill.role+' · '+skill.mana+' mana · '+skill.cooldown+' round cooldown');
   const pin=node('label',undefined,'ab-class-pin'),check=node('input');check.type='checkbox';check.checked=pins.has(skill.id);check.disabled=data.locked;check.dataset.pin=skill.id;check.setAttribute('aria-label','Pin '+skill.name);pin.append(check,document.createTextNode('Pin'));
   check.addEventListener('change',()=>{if(check.checked)input.checked=true;});input.addEventListener('change',()=>{if(!input.checked)check.checked=false;});row.append(input,title,detail,pin);return row;}));
  if(!data.learned.length)$('abyssClassSkills').append(node('p','No acquired skills yet. Your subclass provides its two signature actions immediately.'));
  $('abyssClassSaveSkills').disabled=busy||data.locked;
 }
 function renderGear(){
  $('abyssClassGear').replaceChildren(...data.gear_comparisons.map(g=>{const row=node('tr');[g.name,g.damage_stat+' '+signed(g.damage_delta),signed(g.hp_delta),signed(g.defense_delta),signed(g.mana_delta)].forEach(v=>row.append(node('td',v)));return row;}));
  if(!data.gear_comparisons.length){const row=node('tr'),cell=node('td','Choose a subclass and collect gear to compare candidates.');cell.colSpan=5;row.append(cell);$('abyssClassGear').append(row);}
 }
 function render(){const active=data.catalog.find(c=>c.id===data.state.class||c.subclasses.some(s=>s.id===data.state.selected));
  $('abyssClassContent').hidden=false;$('abyssClassRoster').replaceChildren(...data.catalog.map((c,i)=>{const b=node('button'),art=node('span',undefined,'ab-class-mini');b.type='button';b.dataset.class=c.id;b.setAttribute('aria-pressed','false');art.style.backgroundPosition='0% '+(i*20)+'%';art.setAttribute('aria-hidden','true');b.append(art,node('span',c.name));b.addEventListener('click',()=>chooseClass(c.id));return b;}));
  chooseClass(active?active.id:(shownClass?shownClass.id:data.catalog[0].id));
  $('abyssClassWeakness').textContent=data.weakness;
  $('abyssClassLegacy').disabled=busy||data.locked||(!data.state.selected&&!data.state.class);
  const activeSub=active&&active.subclasses.find(s=>s.id===data.state.selected);
  status(data.locked?'Build locked for this run. Bank or end the run to switch.':activeSub?active.name+' / '+activeSub.name+' active. Switching between runs is free.':active?active.name+' foundation active. Earn class XP from Abyss monster kills.':'Your original build is active. Inspect a class to choose a new combat style.');
 }
 async function load(){if(busy)return;busy=true;status('Loading your build…');try{data=await request();render();}catch(err){status(err.message);}finally{busy=false;if(data){renderSubclass();$('abyssClassLegacy').disabled=data.locked||(!data.state.selected&&!data.state.class);}}}
 async function save(profile){if(busy||!data)return;busy=true;root.setAttribute('aria-busy','true');$('abyssClassChoose').disabled=true;$('abyssClassSaveSkills').disabled=true;
  try{data=await request({selected:profile===null||subclass().id===shownClass.id?'':subclass().id,class:profile===null?'':shownClass.id,revision:data.state.revision,...(profile?{profile}: {})});render();status('Build saved. '+((data.state.selected||data.state.class)?'Your class graphic and skills are ready for the next run.':'Your original build is active.'));}
  catch(err){status(err.message);}finally{busy=false;root.removeAttribute('aria-busy');if(data){renderSubclass();$('abyssClassLegacy').disabled=data.locked||(!data.state.selected&&!data.state.class);}}
 }
 window.AbyssClassWorkbench={refresh:load,saveTalents:async function(body){if(busy||!data)return;busy=true;try{data=await request({...body,revision:data.state.revision});render();status('Talent build saved. Your choices apply to the next fight.');}catch(err){status(err.message);}finally{busy=false;renderSubclass();}}};
 $('abyssClassChoose').addEventListener('click',()=>save());$('abyssClassLegacy').addEventListener('click',()=>save(null));$('abyssSubclass').addEventListener('change',renderSubclass);$('abyssClassReload').addEventListener('click',load);
 $('abyssClassSaveSkills').addEventListener('click',()=>{const skills=Array.from(root.querySelectorAll('[data-equip]:checked'),e=>e.value),pins=Array.from(root.querySelectorAll('[data-pin]:checked'),e=>e.dataset.pin);if(skills.length>data.capacity){status('Choose at most '+data.capacity+' acquired skills. Your signature actions use additional slots.');return;}save({skills,pins});});
 // Existing action responses may begin/end a run in place. Refresh only when its state changes.
 let lastRun=typeof window.inRun==='boolean'?window.inRun:null;
 const timer=setInterval(()=>{if(document.hidden||!data)return;const current=typeof window.inRun==='boolean'?window.inRun:null;if(current!==null&&current!==lastRun){lastRun=current;load();}},2500);
 window.addEventListener('pagehide',()=>clearInterval(timer),{once:true});load();
})();
