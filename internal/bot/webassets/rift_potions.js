(function(){
 'use strict';
 function validate(data){
  if(data?.ok!==true||!Array.isArray(data.potions)||data.potions.length>100)throw Error('Potion inventory could not be confirmed.');
  const ids=new Set();
  for(const p of data.potions){
   if(!p||typeof p.id!=='string'||!p.id||p.id.length>100||ids.has(p.id)||typeof p.name!=='string'||!Number.isSafeInteger(p.count)||p.count<=0||!((Number.isFinite(p.heal_hp)&&p.heal_hp>1&&p.heal_fraction===undefined)||(Number.isFinite(p.heal_fraction)&&p.heal_fraction>0&&p.heal_fraction<=1&&p.heal_hp===undefined)))throw Error('Potion inventory could not be confirmed.');
   ids.add(p.id);
  }
  return data.potions;
 }
 function create({api,practice}){
  const panel=document.getElementById('rift-potions'),select=document.getElementById('rift-potion-select'),use=document.getElementById('rift-potion-use'),refresh=document.getElementById('rift-potion-refresh'),status=document.getElementById('rift-potion-status');
  if(!panel)return null;panel.hidden=!!practice;
  let items=[],run=null,playing=false,loaded=false,loading=false,pending='',inFlight=false,uncertain=false,controller=null;
  const selected=()=>items.find(item=>item.id===select.value);
  function reason(){
   if(uncertain)return 'Recover the saved expedition to confirm potion use.';
   if(inFlight)return 'Confirming potion use…';
   if(pending)return 'Potion queued for the next combat update.';
   if(loading)return 'Loading owned potions…';
   if(!loaded)return 'Open this panel to load owned healing potions.';
   if(!selected())return 'No supported healing potions in your Abyss inventory.';
   if(!playing||run?.paused||run?.status!=='fighting'||!(run.player.hp>0))return 'Resume combat to use a potion.';
   if(run.player.hp>=run.player.max_hp)return 'Health is full. No potion will be consumed.';
   const remaining=run.skill_timers?.healing_potion||0;
   if(remaining>0)return 'Potion ready in '+Math.ceil(remaining)+' combat seconds.';
   return '';
  }
  function render(){const why=reason();use.disabled=!!why;select.disabled=loading||!!pending||inFlight||uncertain;refresh.disabled=loading||!!pending||inFlight;const message=why||'Ready · consumes one owned potion.';if(status.textContent!==message)status.textContent=message;}
  async function load(){
   if(practice||loading||inFlight)return;
   loading=true;controller=new AbortController();const timeout=setTimeout(()=>controller?.abort(),10000);render();
   try{
    const url=new URL(api,location.href);url.searchParams.set('inventory','potions');
    const response=await fetch(url,{credentials:'same-origin',cache:'no-store',signal:controller.signal});
    if(!response.ok)throw Error('Potion inventory unavailable. Refresh to try again.');
    const next=validate(await response.json()),previous=select.value;items=next;loaded=true;select.replaceChildren();
    for(const item of items){const option=document.createElement('option');option.value=item.id;option.textContent=item.name+' ×'+item.count+' · heals '+(item.heal_fraction?Math.round(item.heal_fraction*100)+'% max HP':item.heal_hp+' HP');select.append(option);}
    if(items.some(item=>item.id===previous))select.value=previous;
   }catch(error){loaded=false;items=[];select.replaceChildren();status.textContent=error.name==='AbortError'?'Potion inventory timed out. Refresh to try again.':error.message;}
   finally{clearTimeout(timeout);controller=null;loading=false;if(loaded)render();else{use.disabled=true;select.disabled=true;refresh.disabled=false;}}
  }
  panel.addEventListener('toggle',()=>{if(panel.open&&!loaded&&!loading)void load();});refresh.onclick=()=>void load();select.onchange=render;
  use.onclick=()=>{if(reason())return;pending=selected().id;render();};
  window.addEventListener('pagehide',()=>{controller?.abort();pending='';playing=false;});
  render();
  return {
   update(value,active){run=value;playing=active;render();},
   cancel(){pending='';playing=false;render();},
   take(){if(!pending)return '';const id=pending;pending='';if(reason()) {render();return '';}inFlight=true;render();return id;},
   finish(ok){inFlight=false;uncertain=!ok;render();if(ok)void load();},
   recover(){pending='';inFlight=false;uncertain=false;render();if(loaded||panel.open)void load();}
  };
 }
 window.RiftPotions={create,validate};
})();
