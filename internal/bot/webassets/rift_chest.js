(function(){
  'use strict';
  function createChest(loadArt){
    let key='',status='',seenClear=false,chest=null,art=null,pending=null,attemptedKey='';
    function prepare(){
      if(art||pending||attemptedKey===key)return;
      attemptedKey=key;
      pending=Promise.resolve().then(loadArt).then(value=>{art=value;}).catch(()=>{}).finally(()=>{pending=null;});
    }
    function observe(run,replay,now){
      const next=run?[run.id,run.level?.id,run.room].join(':'):'';
      if(next!==key){key=next;status='';seenClear=false;chest=null;}
      if(!run||run.practice){chest=null;return false;}
      // Optional art never participates in critical startup readiness.
      if(run.status==='cleared'||run.status==='fighting'&&!run.paused&&!replay)prepare();
      const loot=run.drops?.some(d=>!d.banked&&d.collected&&d.mission===run.level?.id&&d.tier===run.room+1&&(d.gold>0||d.gear));
      const opening=run.status==='cleared'&&status==='fighting'&&!seenClear&&!replay&&!!loot;
      if(run.status==='cleared'&&loot){
        if(!chest){
          const p=run.player,arena=run.level.rooms[run.room],walls=[...(arena.obstacles||[]),...(arena.high_cover||[]),...(arena.cover||[]).filter(c=>c.material==='stone'||c.hp>0)];
          const offsets=p.elevation?[0]:[58,-58,96,-96,140,-140,0];
          const x=offsets.map(offset=>Math.max(70,Math.min(1530,p.x+offset))).find(x=>(!arena.exit||Math.abs(x-arena.exit.x)>=80)&&!walls.some(w=>x+24>w.x&&x-24<w.x+w.w&&p.y+10>w.y&&p.y-10<w.y+w.h));
          if(x!==undefined)chest={x,y:p.y,elevation:p.elevation||0,started:opening?now:null};
        }
        if(replay&&chest)chest.started=null;
        seenClear=true;
      }else chest=null;
      status=run.status;
      return opening&&!run.paused&&!!chest;
    }
    return {observe,image:()=>art,ready:()=>pending||Promise.resolve(),frame(now,still){return chest?{...chest,index:still||chest.started===null?5:Math.min(5,Math.max(0,Math.floor((now-chest.started)/120)))}:null;}};
  }
  if(typeof module==='object'&&module.exports){module.exports={createChest};return;}
  const source=document.getElementById('rift-app').dataset.chest;
  window.RiftChest=createChest(()=>new Promise((resolve,reject)=>{
    const img=new Image();img.onload=async()=>{try{await img.decode();if(img.naturalWidth!==2172||img.naturalHeight!==724)throw Error('Invalid chest sheet');resolve(img);}catch(error){reject(error);}};
    img.onerror=()=>reject(Error('Chest art unavailable'));img.src=source;
  }));
})();
