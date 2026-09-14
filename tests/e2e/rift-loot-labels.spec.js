const {test,expect}=require('@playwright/test');

test('nearby loot labels fit without overlaps while pickup coordinates stay fixed',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const result=await page.evaluate(()=>{
  const drops=Array.from({length:20},(_,i)=>({id:String(i),x:10,y:80,gear:{Rarity:3}}));const before=JSON.stringify(drops);
  const labels=window.RiftLoot.floorLabels(drops,0);let overlaps=0;
  for(let i=0;i<labels.length;i++)for(let j=i+1;j<labels.length;j++)if(Math.abs(labels[i].x-labels[j].x)<114&&Math.abs(labels[i].y-labels[j].y)<17)overlaps++;
  const filtered=window.RiftLoot.floorLabels([{...drops[0],collected:true},{...drops[1],banked:true},{...drops[2],x:2000},{...drops[3],gear:null}],0);
  return {count:labels.length,overlaps,unchanged:before===JSON.stringify(drops),inside:labels.every(l=>l.x>=57&&l.x<=903&&l.y>=0&&l.y+17<=540),moved:labels.some(l=>l.moved),filtered:filtered.length};
 });
 expect(result).toEqual({count:20,overlaps:0,unchanged:true,inside:true,moved:true,filtered:0});
});
