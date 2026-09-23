const {test,expect}=require('@playwright/test');
test('bestiary displays measured boss room clear and omits missing timing',async({page})=>{
 let name;
 await page.route('**/api/abyss/rift',async route=>{
  const response=await route.fetch(),data=await response.json();
  if(route.request().method()==='GET'&&data.run){const boss=data.bestiary.find(unit=>unit.kind==='boss');name=boss.name;data.run.monster_records={...data.run.monster_records,[boss.art_key]:{first_seen_ms:100000,defeats:2,fastest_clear_seconds:8.125}};}
  await route.fulfill({response,json:data});
 });
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.evaluate(name=>window.RiftBestiary.openBoss(name,document.getElementById('rift-start')),name);
 await expect(page.locator('#rift-monster-record')).toContainText('Fastest boss room clear: 8.13 s');
 await page.evaluate(()=>window.RiftBestiary.update({monster_records:{}}));
 await expect(page.locator('#rift-monster-record')).not.toContainText('Fastest boss room clear');
});
