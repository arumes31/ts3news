const {test,expect}=require('@playwright/test');
test('hazard defeat advice matches saved cause and jumpability',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();
 for(const [kind,jumpable,expected] of [['fire',true,'or jump while it is active'],['ice',true,'does not prevent slowing'],['poison',true,'does not prevent slowing'],['thorns',true,'does not prevent slowing'],['void',false,'does not stop the pull'],['rune',false,'cannot be jumped'],['radiant',true,'Guard reduces damage'],['collapse',false,'Jumping does not evade collapse damage']]){
  await page.evaluate(({run,kind,jumpable})=>{run.status='defeated';run.defeated_by_hazard={kind,jumpable};run.last_encounter={mission:1,room:0,room_name:'Gate',outcome:'defeated',seconds:12,player_hp:0,player_max_hp:100,defeated_by_hazard:run.defeated_by_hazard};RiftOnboarding.update(run);RiftHUD.updateLastEncounter(run);},{run,kind,jumpable});
  await expect(page.locator('#rift-hazard-defeat-hint')).toBeVisible();await expect(page.locator('#rift-hazard-defeat-hint')).toContainText(expected);await expect(page.locator('#rift-last-encounter-stats')).toContainText(expected);
 }
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.locator('#rift-defeat-guide').screenshot({path:'test-results/hazard-defeat-guide.png'});
 await page.evaluate(run=>{run.status='defeated';delete run.defeated_by_hazard;RiftOnboarding.update(run);},run);await expect(page.locator('#rift-hazard-defeat-hint')).toBeHidden();
});

test('hazard defeat protocol rejects malformed saved causes',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>[undefined,{kind:'ice',jumpable:true},{kind:'collapse',jumpable:false},null,{kind:'ice',jumpable:'true'},{kind:'',jumpable:true}].map(source=>{const copy=structuredClone(data);copy.run.defeated_by_hazard=source;try{RiftProtocol.validate(copy,'GET');return true;}catch(_){return false;}}),data)).toEqual([true,true,true,false,false,false]);
});
