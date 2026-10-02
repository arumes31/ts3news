(function(){
  'use strict';
  const $=id=>document.getElementById(id),key='riftLoadoutPresets';
  let presets=[],skills=[],blocked=()=>true,previewVersion=0,animationFrame=0;
  function cancelSkillPreview(){previewVersion++;cancelAnimationFrame(animationFrame);animationFrame=0;document.querySelectorAll('.rift-skill-animation-status').forEach(status=>{if(status.textContent.startsWith('Playing')||status.textContent.startsWith('Loading'))status.textContent='Preview stopped. Replay at any time.';});window.RiftAudio.cancelPreview();}

  try{const saved=JSON.parse(localStorage.getItem(key));if(Array.isArray(saved))presets=saved.filter(p=>p&&typeof p.id==='string'&&typeof p.name==='string'&&Array.isArray(p.skills)&&p.skills.length<=3&&p.skills.every(id=>typeof id==='string'&&id.length<=120)).slice(0,10).map(p=>({...p,name:p.name.slice(0,40)}));}catch(_){}
  const section=document.createElement('div');section.id='rift-loadout-presets';section.hidden=true;section.setAttribute('role','group');section.setAttribute('aria-label','Skill presets');
  section.innerHTML='<h4>Skill presets</h4><label for="rift-preset-list">Saved preset <select id="rift-preset-list"><option value="">Choose a preset</option></select></label><label for="rift-loadout-name">Preset name <input id="rift-loadout-name" maxlength="40" placeholder="e.g. Guardian hunter"></label><div><button type="button" id="rift-save-loadout">Save current skills</button><button type="button" id="rift-rename-loadout">Rename</button><button type="button" id="rift-delete-loadout">Delete preset</button></div><p id="rift-loadout-review"></p><button type="button" id="rift-apply-loadout">Apply reviewed skills</button><p id="rift-loadout-status" role="status"></p>';
  $('rift-loadout').after(section);
  const reorder=document.createElement('div');reorder.id='rift-loadout-order';reorder.setAttribute('role','group');reorder.setAttribute('aria-label','Reorder expedition skills');$('rift-loadout').after(reorder);
  const empty=document.createElement('p');empty.id='rift-empty-loadout';empty.hidden=true;empty.textContent='No optional skills selected. Basic attacks, jumping and guarding remain available. Learned class abilities are separate from these slots.';$('rift-loadout').after(empty);
  const transfer=document.createElement('details');transfer.innerHTML='<summary>Import or export a preset</summary><label for="rift-preset-json">Preset JSON <textarea id="rift-preset-json" maxlength="2000" rows="5" spellcheck="false"></textarea></label><button type="button" id="rift-export-loadout">Export selected preset</button><button type="button" id="rift-import-loadout">Import for review</button><p>Copy the JSON to transfer a named preset. Import saves it locally for review; it does not equip skills.</p>';section.append(transfer);
  const glossary=document.createElement('details');glossary.id='rift-skill-glossary';glossary.innerHTML='<summary>Skill reference</summary><p><a href="/abyss/rift?practice=skills">Try equipped abilities in the safe skill testing lane</a></p><p>Current Abyss skills. An active expedition keeps the build it started with. Character, class, gear and learned-skill changes apply when you start a new expedition, so combat stays consistent throughout the current run.</p><p id="rift-ultimate-ownership"></p><p id="rift-ultimate-selection"></p><p id="rift-class-cost-reference" hidden></p><p>Ability markers: ＋ builder · ◆ finisher · ★ ultimate.</p><label for="rift-glossary-search">Find a skill <input id="rift-glossary-search" type="search" maxlength="80"></label><p id="rift-glossary-count" role="status"></p><div id="rift-glossary-entries"></div>';section.after(glossary);
  function filterGlossary(){const query=$('rift-glossary-search').value.trim().toLocaleLowerCase(),entries=Array.from($('rift-glossary-entries').children);let count=0;entries.forEach(entry=>{entry.hidden=!entry.dataset.search.includes(query);if(!entry.hidden)count++;});$('rift-glossary-count').textContent=count?count+' of '+entries.length+' skills':'No matching skills.';}
  $('rift-glossary-search').addEventListener('input',()=>{cancelSkillPreview();filterGlossary();});
  glossary.addEventListener('toggle',()=>{if(!glossary.open)cancelSkillPreview();});
  document.addEventListener('visibilitychange',()=>{if(document.hidden)cancelSkillPreview();});
  window.addEventListener('pagehide',cancelSkillPreview);
  const slots=()=>Array.from($('rift-loadout').querySelectorAll('select'));
  const chosen=()=>presets.find(p=>p.id===$('rift-preset-list').value);
  const name=id=>id?(skills.find(skill=>skill.id===id)?.name||'Unavailable skill: '+id):'None';
  const invalid=p=>p.skills.some(id=>id&&!skills.some(skill=>skill.id===id))||new Set(p.skills.filter(Boolean)).size!==p.skills.filter(Boolean).length||p.skills.slice(slots().length).some(Boolean);
  function write(next){try{localStorage.setItem(key,JSON.stringify(next));presets=next;return true;}catch(_){$('rift-loadout-status').textContent='Could not save presets in this browser.';return false;}}
  function options(id=''){$('rift-preset-list').replaceChildren(new Option('Choose a preset',''),...presets.map(p=>new Option(p.name,p.id)));$('rift-preset-list').value=id;refresh();}
  function refresh(){
    const p=chosen(),locked=blocked();
    $('rift-save-loadout').disabled=locked||presets.length>=10;
    $('rift-rename-loadout').disabled=locked||!p;$('rift-delete-loadout').disabled=locked||!p;
    $('rift-apply-loadout').disabled=locked||!p||invalid(p);
    $('rift-import-loadout').disabled=locked||presets.length>=10;$('rift-export-loadout').disabled=!p;
    empty.hidden=slots().some(slot=>slot.value);
    reorder.querySelectorAll('button').forEach(button=>button.disabled=locked);
    $('rift-glossary-entries').querySelectorAll('[data-optional-skill]').forEach(entry=>{const index=slots().findIndex(slot=>slot.value===entry.dataset.optionalSkill);entry.querySelector('.rift-skill-slot').textContent=index<0?'Not selected':'Slot '+(index+1);});
    $('rift-loadout-review').textContent=p?slots().map((slot,i)=>'Slot '+(i+1)+': '+name(slot.value)+' → '+name(p.skills[i]||'')).join(' · ')+(invalid(p)?' Cannot apply: unavailable or duplicate skills.':' Applying changes only these expedition skill slots.'):'Save up to 10 presets in this browser. Empty slots keep basic attacks and class abilities available.';
  }
  $('rift-preset-list').addEventListener('change',()=>{$('rift-loadout-name').value=chosen()?.name||'';refresh();});
  $('rift-loadout').addEventListener('change',refresh);
  $('rift-save-loadout').onclick=()=>{if(blocked()||presets.length>=10)return;const title=$('rift-loadout-name').value.trim();if(!title){$('rift-loadout-status').textContent='Enter a preset name.';return;}const p={id:crypto.randomUUID(),name:title,skills:slots().map(slot=>slot.value)};if(invalid(p)){ $('rift-loadout-status').textContent='Choose each owned skill only once.';return;}if(write([...presets,p])){options(p.id);$('rift-loadout-status').textContent='Preset saved locally.';}};
  $('rift-rename-loadout').onclick=()=>{const p=chosen(),title=$('rift-loadout-name').value.trim();if(blocked()||!p)return;if(!title){$('rift-loadout-status').textContent='Enter a preset name.';return;}if(write(presets.map(item=>item.id===p.id?{...item,name:title}:item))){options(p.id);$('rift-loadout-status').textContent='Preset renamed.';}};
  $('rift-delete-loadout').onclick=()=>{const p=chosen();if(blocked()||!p)return;if(write(presets.filter(item=>item.id!==p.id))){options();$('rift-loadout-status').textContent='Preset deleted. Current skills and inventory are unchanged.';}};
  $('rift-apply-loadout').onclick=()=>{const p=chosen();if(blocked()||!p||invalid(p))return;slots().forEach((slot,i)=>{slot.value=p.skills[i]||'';});refresh();$('rift-loadout-status').textContent='Preset applied to the next expedition.';};
  $('rift-export-loadout').onclick=()=>{const p=chosen();if(!p)return;$('rift-preset-json').value=JSON.stringify({version:1,name:p.name,skills:p.skills},null,2);$('rift-preset-json').focus();$('rift-preset-json').select();$('rift-loadout-status').textContent='Preset JSON ready to copy.';};
  $('rift-import-loadout').onclick=()=>{
    if(blocked()||presets.length>=10)return;
    try{
      const source=$('rift-preset-json').value;if(source.length>2000)throw new Error('Preset JSON is too long.');
      const p=JSON.parse(source);
      if(!p||p.version!==1||typeof p.name!=='string'||!p.name.trim()||p.name.length>40||!Array.isArray(p.skills)||p.skills.length>3||!p.skills.every(id=>typeof id==='string'&&id.length<=120))throw new Error('Use a version 1 preset with a name and up to three skill IDs.');
      if(invalid(p))throw new Error('Cannot import: every skill must be owned, fit the available slots and appear only once.');
      const imported={id:crypto.randomUUID(),name:p.name.trim(),skills:p.skills};
      if(write([...presets,imported])){options(imported.id);$('rift-loadout-name').value=imported.name;$('rift-loadout-status').textContent='Preset imported. Review the changes before applying.';}
    }catch(error){$('rift-loadout-status').textContent=error instanceof SyntaxError?'Preset JSON is invalid.':error.message;}
  };
  window.RiftLoadouts={init(build,isBlocked){
    cancelSkillPreview();
    $('rift-class-cost-reference').hidden=build.class!=='voidwalker';$('rift-class-cost-reference').textContent='Voidwalker charged finishers spend 5% of maximum health, capped to leave at least 1 HP. Finishers without charges spend no health.';
    const owned=build.owned_ultimates;
    $('rift-ultimate-ownership').textContent=Array.isArray(owned)?(owned.length?'Owned ultimates: '+owned.join(', '):'No ultimates owned.'):'Full ultimate ownership is unavailable in this older build snapshot.';
    $('rift-ultimate-selection').textContent=build.ultimate?'Selected for Brawl: '+build.ultimate.name+' · '+build.ultimate.cost+' MP · '+build.ultimate.cooldown+'s cooldown. Uses its own slot.':'No active ultimate selected for Brawl. Activate an owned ultimate in Abyss, then start a new expedition.';
    skills=build.skills;blocked=isBlocked;section.hidden=false;reorder.replaceChildren();
    $('rift-glossary-entries').replaceChildren();
    const entries=[...skills.map(skill=>({skill,category:'Optional skill'})),...(build.signatures||[]).map(skill=>({skill,category:skill.role==='builder'?'Class builder':'Class finisher'})),...(build.ultimate?[{skill:build.ultimate,category:'Ultimate'}]:[])];
    for(const {skill,category} of entries){
      const entry=document.createElement('article'),title=document.createElement('strong'),stats=document.createElement('p'),description=document.createElement('p'),slot=document.createElement('small');
      const catalogIcon=document.createElement('span');catalogIcon.className='rift-catalog-icon';catalogIcon.setAttribute('aria-hidden','true');catalogIcon.textContent=skill.icon||'✦';
      entry.dataset.skill=skill.id;entry.dataset.search=(skill.name+' '+category+' '+skill.kind).toLocaleLowerCase();title.textContent=skill.name;
      stats.textContent=category+' · '+skill.cost+' MP · '+skill.cooldown+'s cooldown · Effect: '+skill.kind;
      description.className='rift-skill-description';description.textContent=window.RiftAbilities.describe(skill,build);entry.dataset.search+=' '+description.textContent.toLocaleLowerCase();
      slot.className='rift-skill-slot';if(category==='Optional skill')entry.dataset.optionalSkill=skill.id;else slot.textContent='Separate from optional skill slots';
      const sound=document.createElement('button'),soundStatus=document.createElement('p');
      sound.type='button';sound.className='rift-skill-sound';sound.textContent='Preview '+skill.name+' sound';
      soundStatus.className='rift-skill-sound-status';soundStatus.setAttribute('role','status');
      sound.onclick=async()=>{
        cancelSkillPreview();const version=previewVersion;sound.disabled=true;
        $('rift-glossary-entries').querySelectorAll('.rift-skill-sound-status').forEach(status=>status.textContent='');
        try{
          const played=await window.RiftAudio.preview('effects',skill.kind);
          if(version===previewVersion&&sound.isConnected&&glossary.open)soundStatus.textContent=played?(window.RiftAudio.effects===0?'Effects volume is zero. Raise it in sound settings to hear this skill.':'Previewing '+skill.name+'. Uses your current sound mix.'):(window.RiftAudio.muted?'Sound is muted. Unmute to preview.':'Audio is unavailable or blocked by the browser.');
        }catch(_){if(version===previewVersion&&sound.isConnected)soundStatus.textContent='Audio is unavailable or blocked by the browser.';}
        finally{sound.disabled=false;}
      };
      const animation=document.createElement('button'),preview=document.createElement('canvas'),animationStatus=document.createElement('p');
      animation.type='button';animation.className='rift-skill-animation';animation.textContent='Preview '+skill.name+' animation';
      preview.width=320;preview.height=160;preview.hidden=true;preview.className='rift-skill-animation-canvas';preview.setAttribute('role','img');preview.setAttribute('aria-label',skill.name+' cast and effect artwork preview');
      animationStatus.setAttribute('role','status');animationStatus.className='rift-skill-animation-status';
      animation.onclick=async()=>{
        cancelSkillPreview();const version=previewVersion;
        $('rift-glossary-entries').querySelectorAll('.rift-skill-animation-canvas').forEach(canvas=>canvas.hidden=true);
        $('rift-glossary-entries').querySelectorAll('.rift-skill-animation-status').forEach(status=>status.textContent='');
        animation.disabled=true;animationStatus.textContent='Loading animation artwork…';
        try{await window.RiftRenderer.prepareEffects();}
        catch(_){if(version===previewVersion&&preview.isConnected&&glossary.open)animationStatus.textContent='Could not load animation artwork. Select Preview to try again.';return;}
        finally{animation.disabled=false;}
        if(version!==previewVersion||document.hidden||!preview.isConnected||!glossary.open)return;
        const start=performance.now();preview.hidden=false;
        const draw=now=>{
          if(version!==previewVersion||document.hidden||!preview.isConnected||!glossary.open)return;
          const still=window.RiftRenderer.drawSkillPreview(preview,skill,build,now-start);
          animationStatus.textContent=still?'Static preview: reduced motion is enabled.':now-start>=750?'Animation complete. Replay at any time.':'Playing cast and effect artwork. Sound is previewed separately.';
          if(!still&&now-start<750)animationFrame=requestAnimationFrame(draw);else animationFrame=0;
        };
        draw(start);
      };
      entry.append(catalogIcon,title,stats,description,slot,animation,preview,animationStatus,sound,soundStatus);$('rift-glossary-entries').append(entry);
    }
    filterGlossary();
    for(let index=0;index<slots().length-1;index++){
      const button=document.createElement('button');button.type='button';button.textContent='Swap slots '+(index+1)+' and '+(index+2);
      button.onclick=()=>{if(blocked())return;const current=slots(),value=current[index].value;current[index].value=current[index+1].value;current[index+1].value=value;refresh();$('rift-loadout-status').textContent='Swapped skill slots '+(index+1)+' and '+(index+2)+'.';};
      reorder.append(button);
    }
    options();
  },refresh,openReference(){glossary.open=true;(window.matchMedia('(any-pointer: coarse)').matches?glossary.querySelector('summary'):$('rift-glossary-search')).focus();glossary.scrollIntoView({block:'start'});}};
})();
