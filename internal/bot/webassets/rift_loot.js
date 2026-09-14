(function(){
  'use strict';
  const $=id=>document.getElementById(id),format=new Intl.NumberFormat(undefined,{maximumFractionDigits:0});
  const count=(n,noun)=>format.format(n)+' '+noun+(n===1?'':'s');
  const put=(node,value)=>{if(node.textContent!==value)node.textContent=value;};
  const create=(tag,value,parent)=>{const node=document.createElement(tag);node.textContent=value;parent.append(node);return node;};
  const normalize=value=>String(value).normalize('NFD').replace(/[\u0300-\u036f]/g,'').toLowerCase();
  const slots={weapon:0,mainhand:0,offhand:1,ranged:3,head:4,helmet:4,chest:5,armor:5,feet:6,boots:6,hands:7,gloves:7,ring:12,finger1:12,finger2:12,neck:13,amulet:13,relic:15,artifact:15};
  const icon=slot=>slots[String(slot).toLowerCase()]??9;
  let rarities=new Map(),run=null,bagKey='',receiptKey='';
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
    const groups=new Map();run.banked_items.forEach(name=>groups.set(name,(groups.get(name)||0)+1));
    const query=normalize($('rift-receipt-search').value.trim());
    const matches=[...groups].filter(([name])=>normalize(name).includes(query)).sort(([a],[b])=>a.localeCompare(b));
    $('rift-receipt-list').replaceChildren();
    matches.slice(0,200).forEach(([name,n])=>create('li',name+(n>1?' × '+format.format(n):''),$('rift-receipt-list')));
    $('rift-receipt-empty').hidden=matches.length>0;
    $('rift-receipt-empty').textContent=run.banked_items.length?'No banked items match this search.':'No gear was banked.';
    $('rift-receipt-limit').hidden=matches.length<=200;
    $('rift-receipt-limit').textContent='Showing 200 of '+format.format(matches.length)+' matching item names. Search to narrow the receipt.';
    put($('rift-receipt-total'),format.format(run.banked_gold)+' gold · '+count(run.banked_items.length,'item')+' safely banked');
  }
  function update(value){
    run=value;
    const items=run.drops.filter(d=>d.collected&&!d.banked&&d.gear);
    put($('rift-gold'),format.format(run.gold));put($('rift-banked'),format.format(run.banked_gold)+' gold · '+count(run.banked_items.length,'item'));
    put($('rift-loot-count'),count(items.length,'item')+' pending');
    const pending=run.drops.filter(d=>!d.banked);put($('rift-checkpoint-total'),format.format(pending.reduce((sum,d)=>sum+d.gold,0))+' gold · '+count(pending.filter(d=>d.gear).length,'item')+' ready to bank');
    const floor=run.drops.filter(d=>!d.collected&&!d.banked).length;put($('rift-floor-loot'),floor?count(floor,'drop')+' still on the battlefield':'All available drops collected');
    const key=run.id+':'+JSON.stringify(items.map(d=>[d.id,d.gear.ID]));if(key!==bagKey){bagKey=key;bag();}
    const next=JSON.stringify([run.id,run.banked_gold,run.banked_items]);
    $('rift-receipt').hidden=!run.banked_gold&&!run.banked_items.length&&!['defeated','banked','complete'].includes(run.status);
    if(next!==receiptKey){receiptKey=next;receipt();$('rift-receipt-copy-status').textContent='';}
  }
  $('rift-loot-sort').onchange=bag;$('rift-receipt-search').oninput=receipt;
  $('rift-copy-receipt').onclick=async()=>{
    if(!run)return;
    const names=new Map();run.banked_items.forEach(name=>names.set(name,(names.get(name)||0)+1));
    const lines=['Rift Brawl — banked rewards',format.format(run.banked_gold)+' gold',count(run.banked_items.length,'item'),...[...names].map(([name,n])=>name+(n>1?' × '+format.format(n):''))];
    try{await navigator.clipboard.writeText(lines.join('\n'));$('rift-receipt-copy-status').textContent='Receipt copied.';}catch(_){$('rift-receipt-copy-status').textContent='Copy is unavailable. Select the receipt text to copy it.';}
  };
  window.RiftLoot={icon,init(values){rarities=new Map(values.filter(v=>v&&Number.isInteger(v.value)&&typeof v.name==='string'&&/^#[0-9a-f]{6}$/i.test(v.color)).map(v=>[v.value,v]));bagKey='';},update};
})();
