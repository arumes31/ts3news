(function(){
  'use strict';
  const $=id=>document.getElementById(id),format=new Intl.NumberFormat(undefined,{maximumFractionDigits:0});
  const count=(n,noun)=>format.format(n)+' '+noun+(n===1?'':'s');
  const put=(node,value)=>{if(node.textContent!==value)node.textContent=value;};
  const create=(tag,value,parent)=>{const node=document.createElement(tag);node.textContent=value;parent.append(node);return node;};
  const normalize=value=>String(value).normalize('NFD').replace(/[\u0300-\u036f]/g,'').toLowerCase();
  const slots={weapon:0,mainhand:0,offhand:1,ranged:3,head:4,helmet:4,chest:5,armor:5,feet:6,boots:6,hands:7,gloves:7,ring:12,finger1:12,finger2:12,neck:13,amulet:13,relic:15,artifact:15};
  const legendary=drop=>!!drop.gear&&rarities.get(drop.gear.Rarity)?.legendary===true;
  function floorLabels(drops,camera,width=960,height=540){
    const labels=[],half=57,row=21,minX=half+4,maxX=width-half-4;
    const fits=(x,y)=>labels.every(label=>Math.abs(label.x-x)>=118||Math.abs(label.y-y)>=row);
    for(const drop of drops){
      if(!drop.gear||drop.collected||drop.banked||drop.x-camera < -36||drop.x-camera > width+36)continue;
      const rare=legendary(drop),preferredX=Math.max(minX,Math.min(maxX,drop.x-camera)),preferredY=Math.max(4,Math.min(height-21,drop.y-8-(rare?52:33)));
      const columns=[preferredX];for(let x=minX;x<=maxX;x+=122)if(Math.abs(x-preferredX)>1)columns.push(x);
      columns.sort((a,b)=>Math.abs(a-preferredX)-Math.abs(b-preferredX));
      let position=null;
      for(const x of columns){
        for(let step=0;step<=Math.ceil(height/row)&&!position;step++){
          for(const y of step?[preferredY-step*row,preferredY+step*row]:[preferredY])if(y>=4&&y<=height-21&&fits(x,y)){position={x,y};break;}
        }
        if(position)break;
      }
      if(position)labels.push({...position,drop,legendary:rare,text:rare?'◆ LEGENDARY':'ABYSS GEAR',moved:position.x!==preferredX||position.y!==preferredY});
    }
    return labels;
  }
  const icon=slot=>slots[String(slot).toLowerCase()]??9;
  let rarities=new Map(),run=null,bagKey='',receiptKey='';
  let pickupIdentity='',collected=new Set(),pickupTimer=0;
  const notices=document.createElement('div');notices.id='rift-pickup-notices';notices.hidden=true;notices.setAttribute('role','status');notices.setAttribute('aria-atomic','true');document.querySelector('.rift-combat-signals').before(notices);
  function pickups(value,replay){
    const identity=[value.id,value.level?.id,value.room].join(':');
    const fresh=identity!==pickupIdentity;
    const picked=value.drops.filter(d=>d.collected&&!d.banked&&!collected.has(d.id));
    collected=new Set(value.drops.filter(d=>d.collected).map(d=>d.id));pickupIdentity=identity;
    if(fresh||replay||!['fighting','cleared'].includes(value.status)){clearTimeout(pickupTimer);delete notices.dataset.confirmed;notices.replaceChildren();notices.hidden=true;return;}
    if(!picked.length)return;
    const gold=picked.reduce((sum,d)=>sum+d.gold,0),gear=picked.filter(d=>d.gear).map(d=>d.gear.Name);
    notices.replaceChildren();
    if(gold>0)create('p','✓ Gold picked up: +'+format.format(gold)+' · unbanked',notices);
    if(gear.length)create('p','✓ Gear picked up: '+gear.slice(0,3).join(', ')+(gear.length>3?' and '+(gear.length-3)+' more':'')+' · unbanked',notices);
    notices.dataset.confirmed='true';
    notices.hidden=!notices.childElementCount;clearTimeout(pickupTimer);pickupTimer=setTimeout(()=>{notices.hidden=true;delete notices.dataset.confirmed;notices.replaceChildren();},4000);
  }
  function bag(){
    if(!run)return;
    const items=run.drops.filter(drop=>drop.collected&&!drop.banked&&drop.gear),sort=$('rift-loot-sort').value;
    if(sort==='rarity')items.sort((a,b)=>(b.gear.Rarity||0)-(a.gear.Rarity||0)||a.gear.Name.localeCompare(b.gear.Name));
    if(sort==='slot')items.sort((a,b)=>a.gear.Slot.localeCompare(b.gear.Slot)||a.gear.Name.localeCompare(b.gear.Name));
    $('rift-loot').replaceChildren();
    if(!items.length)create('li','Your next discovery is out there.',$('rift-loot')).className='rift-empty';
    for(const drop of items){
      const gear=drop.gear,quality=rarities.get(gear.Rarity),li=create('li','',$('rift-loot'));
      li.style.setProperty('--loot-rarity',quality?.color||'#b8cbbb');
      const details=create('details','',li),summary=create('summary','',details),art=create('span','',summary);
      const slot=icon(gear.Slot);art.className='rift-loot-icon';art.setAttribute('aria-hidden','true');art.style.backgroundImage='url("'+$('rift-app').dataset.items+'")';art.style.backgroundPosition=(slot%4*100/3)+'% '+(Math.floor(slot/4)*100/3)+'%';
      create('strong',gear.Name,summary);
      create('small',(quality?.name||'Quality '+(gear.Rarity??'?'))+' · '+gear.Slot+' · Ready to bank',details);
      if(drop.mission&&drop.tier)create('p','Mission '+drop.mission+' · Tier '+drop.tier,details);
      if(gear.found_boss){summary.title='Found: '+gear.found_boss;create('p','Found: '+gear.found_boss,details);}
      const stats=create('dl','',details);
      for(const [name,value] of Object.entries(gear.Stats||{}))if(typeof value==='number'&&value!==0){create('dt',name,stats);create('dd',format.format(value),stats);}
      create('dt','Maximum durability',stats);create('dd',format.format(gear.MaxDurability||0),stats);
      if(gear.Element)create('p','Element: '+gear.Element,details);
    }
  }
  function receipt(){
    if(!run)return;
    const rareOnly=$('rift-receipt-rarity').value==='rare',records=run.banked_loot||[];
    const names=rareOnly?records.filter(item=>rarities.get(item.rarity)?.rare_or_better===true).map(item=>item.name):run.banked_items;
    const unknown=Math.max(0,run.banked_items.length-records.length)+records.filter(item=>!rarities.has(item.rarity)).length;$('rift-receipt-unknown').hidden=!rareOnly||!unknown;put($('rift-receipt-unknown'),count(unknown,'item')+' without recognized rarity data. Choose All banked gear to include them.');
    const groups=new Map();names.forEach(name=>groups.set(name,(groups.get(name)||0)+1));
    const query=normalize($('rift-receipt-search').value.trim());
    const matches=[...groups].filter(([name])=>normalize(name).includes(query)).sort(([a],[b])=>a.localeCompare(b));
    $('rift-receipt-list').replaceChildren();
    matches.slice(0,200).forEach(([name,n])=>create('li',name+(n>1?' × '+format.format(n):''),$('rift-receipt-list')));
    $('rift-receipt-empty').hidden=matches.length>0;
    $('rift-receipt-empty').textContent=run.banked_items.length?'No banked items match these filters.':'No gear was banked.';
    $('rift-receipt-limit').hidden=matches.length<=200;
    $('rift-receipt-limit').textContent='Showing 200 of '+format.format(matches.length)+' matching item names. Search to narrow the receipt.';
    put($('rift-receipt-breakdown'),'Fight loot: '+format.format(run.banked_gold-(run.banked_objective_gold||0))+' gold · Objective bonuses: '+format.format(run.banked_objective_gold||0)+' gold');
    put($('rift-receipt-total'),format.format(run.banked_gold)+' gold · '+count(run.banked_items.length,'item')+' safely banked');
  }
  function update(value,replay=false){
    pickups(value,replay);run=value;
    const items=run.drops.filter(d=>d.collected&&!d.banked&&d.gear);
    put($('rift-gold'),format.format(run.gold));put($('rift-banked'),format.format(run.banked_gold)+' gold · '+count(run.banked_items.length,'item'));
    put($('rift-loot-count'),count(items.length,'item')+' pending');
    const pending=run.drops.filter(d=>!d.banked);put($('rift-checkpoint-total'),format.format(pending.reduce((sum,d)=>sum+d.gold,0))+' gold · '+count(pending.filter(d=>d.gear).length,'item')+' ready to bank');
    const bonus=window.RiftObjectives.pendingGold(run);put($('rift-checkpoint-bonus'),bonus?'Objective bonus: '+format.format(bonus)+' gold (separate from fight loot)':'');
    const legendaryCount=run.drops.filter(d=>!d.collected&&!d.banked&&legendary(d)).length;
    put($('rift-legendary-drops'),legendaryCount?'◆ '+count(legendaryCount,'legendary drop')+' on the battlefield':'No legendary drops on the battlefield');
    const floor=run.drops.filter(d=>!d.collected&&!d.banked).length;put($('rift-floor-loot'),floor?count(floor,'drop')+' still on the battlefield':'All available drops collected');
    put($('rift-pending-hud'),'Unbanked: '+format.format(pending.reduce((sum,d)=>sum+d.gold,0))+' gold · '+pending.filter(d=>d.gear).length+' gear'+(floor?' · '+floor+' uncollected':''));
    let nearest=null,distance=Infinity;
    for(const drop of pending){if(drop.collected)continue;const dx=drop.x-run.player.x,dy=drop.y-run.player.y,squared=dx*dx+dy*dy;if(squared<distance){distance=squared;nearest={dx,dy};}}
    const directions=['→ right','↘ lower right','↓ down','↙ lower left','← left','↖ upper left','↑ up','↗ upper right'];
    put($('rift-nearest-drop'),nearest?'Nearest drop: '+(distance<1?'here':directions[(Math.round(Math.atan2(nearest.dy,nearest.dx)/(Math.PI/4))+8)%8]):'No uncollected drops');
    const key=run.id+':'+JSON.stringify(items.map(d=>[d.id,d.gear.ID]));if(key!==bagKey){bagKey=key;bag();}
    put($('rift-banked-at'),run.banked_at_ms?'Last banked: '+new Date(run.banked_at_ms).toLocaleString():run.banked_gold||run.banked_items.length?'Banking time unavailable for this older receipt.':'No rewards banked yet.');
    const next=JSON.stringify([run.id,run.banked_gold,run.banked_objective_gold,run.banked_items,run.banked_loot]);
    $('rift-receipt').hidden=!run.banked_gold&&!run.banked_items.length&&!['defeated','banked','complete'].includes(run.status);
    if(next!==receiptKey){receiptKey=next;receipt();$('rift-receipt-copy-status').textContent='';}
  }
  $('rift-receipt-rarity').onchange=receipt;$('rift-loot-sort').onchange=bag;$('rift-receipt-search').oninput=receipt;
  $('rift-copy-receipt').onclick=async()=>{
    if(!run)return;
    const names=new Map();run.banked_items.forEach(name=>names.set(name,(names.get(name)||0)+1));
    const lines=['Rift Brawl — banked rewards',format.format(run.banked_gold)+' gold','Fight loot: '+format.format(run.banked_gold-(run.banked_objective_gold||0))+' gold','Objective bonuses: '+format.format(run.banked_objective_gold||0)+' gold',count(run.banked_items.length,'item'),...[...names].map(([name,n])=>name+(n>1?' × '+format.format(n):''))];
    try{await navigator.clipboard.writeText(lines.join('\n'));$('rift-receipt-copy-status').textContent='Receipt copied.';}catch(_){$('rift-receipt-copy-status').textContent='Copy is unavailable. Select the receipt text to copy it.';}
  };
  window.RiftLoot={icon,legendary,floorLabels,banking(state){const messages={pending:'Banking rewards… Inventory delivery is not confirmed yet.',uncertain:'Inventory delivery is unconfirmed. Use Recover expedition to reload saved rewards before continuing.',confirmed:'Reward delivery confirmed. The banked total and receipt are up to date.',reloaded:'Saved reward state loaded. The banked total and receipt show confirmed rewards.'};put($('rift-banking-status'),messages[state]||'');},init(values){rarities=new Map(values.filter(v=>v&&Number.isInteger(v.value)&&typeof v.name==='string'&&/^#[0-9a-f]{6}$/i.test(v.color)).map(v=>[v.value,v]));bagKey='';},update};
})();
