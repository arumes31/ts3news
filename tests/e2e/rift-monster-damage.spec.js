const {test,expect}=require('@playwright/test');
test('bestiary displays recorded health loss and validates damage totals',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json(),unit=data.bestiary.find(u=>u.kind!=='boss');
 const record={first_seen_ms:100000,defeats:2,damage_taken:12.75};
 await page.evaluate(({unit,record})=>{window.RiftBestiary.update({monster_records:{[unit.art_key]:record}});window.RiftBestiary.openMonster(unit.art_key,document.getElementById('rift-start'));},{unit,record});
 const details=page.locator('#rift-monster-record');
 await expect(details).toContainText('Recorded HP lost to this monster: 12.8');await expect(details).toContainText('Older fights may be missing.');
 expect(await page.evaluate(({data,unit,record})=>[undefined,0,12.75,-1,'12',null,Infinity,NaN].map(damage=>{
  data.run.monster_records={[unit.art_key]:{...record,damage_taken:damage}};
  try{window.RiftProtocol.validate(data,'GET');return true;}catch(_){return false;}
 }),{data,unit,record})).toEqual([true,true,true,false,false,false,false,false]);
 await page.evaluate(({unit,record})=>{delete record.damage_taken;window.RiftBestiary.update({monster_records:{[unit.art_key]:record}});},{unit,record});
 await expect(details).not.toContainText('Recorded HP lost');
 await page.evaluate(({unit,record})=>window.RiftBestiary.update({practice:{mode:'boss'},monster_records:{[unit.art_key]:record}}),{unit,record});
 await expect(details).toContainText('Practice does not add encounters, defeats or damage records.');await expect(details).not.toContainText('Recorded HP lost');
});
