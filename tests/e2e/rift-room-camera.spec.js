const {test,expect}=require('@playwright/test');
for(const [scenario,lead] of [['split-defense',430],['spirit',280]])test('authored room framing persists and preserves tracking: '+scenario,async({page})=>{
 await page.goto('/abyss/rift?scenario='+scenario);await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();expect(data.run.level.rooms[data.run.room].camera_lead).toBe(lead);
 const feed=(x,paused=true,boss=false,practice=false)=>page.evaluate(({run,x,paused,boss,practice})=>{run.paused=paused;run.player.x=x;run.player.pose='idle';run.player.pose_time=0;run.player.recoil_x=0;run.player.knockdown=0;run.enemies=boss?[{id:'boss-camera',kind:'boss',x:500,y:410,hp:100,max_hp:100,facing:1,pose:'idle'}]:[];run.room_objective=null;run.events=[];if(practice)run.practice={mode:'movement',arena:{},goal_x:1400};RiftDisplay.cameraSmooth=false;RiftRenderer.snapshot(run,true);},{run:data.run,x,paused,boss,practice});
 const camera=()=>page.evaluate(()=>RiftRenderer.cameraFraming.x);
 await feed(850);await expect.poll(camera).toBe(850-lead);
 await feed(870,false);await page.waitForTimeout(150);expect(await camera()).toBe(850-lead);
 await feed(850,true,true);await expect.poll(()=>page.evaluate(()=>RiftRenderer.cameraFraming.bosses)).toContain('boss-camera');const x=await camera();expect(500-x-84).toBeGreaterThanOrEqual(0);expect(850-x).toBeLessThanOrEqual(896);
 await feed(850,true,false,true);await expect.poll(camera).toBe(500);
 const valid=await page.evaluate(data=>[undefined,260,480,0,259,481,'430',null].map(value=>{const copy=structuredClone(data);copy.run.level.rooms[copy.run.room].camera_lead=value;try{RiftProtocol.validate(copy,'GET');return true;}catch{return false;}}),data);expect(valid).toEqual([true,true,true,false,false,false,false,false]);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();const saved=await(await page.request.get('/api/abyss/rift')).json();expect(saved.run.level.rooms[saved.run.room].camera_lead).toBe(lead);
});
