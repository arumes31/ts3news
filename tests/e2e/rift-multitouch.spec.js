const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true,viewport:{width:1000,height:1400}});
async function setup(page){
 let input=null;const seen=[];
 await page.addInitScript(()=>{window.touchProbe=[];document.addEventListener('pointerdown',event=>{const button=event.target.closest('[data-action]');if(button)window.touchProbe.push({id:event.pointerId,button});},true);});
 await page.route('**/api/abyss/rift*',async route=>{const body=route.request().postDataJSON();if(body?.kind==='step'){input=body.input;seen.push(input);}await route.continue();});
 await page.goto('/abyss/rift?practice=skills');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 const left=page.locator('[data-move=left]');await left.scrollIntoViewIfNeeded();
 const cdp=await page.context().newCDPSession(page);
 return {left,cdp,seen,input:()=>input};
}
// Exercise a deterministic single-pointer release
// using its actual browser-assigned ID, then release capture to check duplicate
// lostpointercapture handling while the other physical touch remains down.
async function releasePointer(page,action){
 await page.evaluate(action=>{
  const index=window.touchProbe.findIndex(record=>record.button.dataset.action===action);
  if(index<0)throw new Error('No active pointer for '+action);
  const {id,button}=window.touchProbe.splice(index,1)[0];
  button.dispatchEvent(new PointerEvent('pointerup',{pointerId:id,pointerType:'touch',bubbles:true}));
  if(button.hasPointerCapture(id))button.releasePointerCapture(id);
 },action);
}
function point(box,id,dx=0){return {id,x:box.x+box.width/2+dx,y:box.y+box.height/2,radiusX:2,radiusY:2,force:1};}

test('releasing one of two fingers on the same control keeps the other held',async({page})=>{
 const {left,cdp,input}=await setup(page),box=await left.boundingBox(),a=point(box,1,-8),b=point(box,2,8);
 await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[a]});
 await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[a,b]});
 await expect.poll(()=>input()?.x).toBe(-1);
 await releasePointer(page,'left');
 await expect(left).toHaveAttribute('aria-pressed','true');await expect.poll(()=>input()?.x).toBe(-1);
 await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
 await expect(left).toHaveAttribute('aria-pressed','false');await expect.poll(()=>input()?.x).toBe(0);
});

test('movement and guard fingers release independently and cancellation clears guard',async({page})=>{
 const {left,cdp,input}=await setup(page),guard=page.locator('[data-bind=guard]');
 const a=point(await left.boundingBox(),1),b=point(await guard.boundingBox(),2);
 await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[a]});
 await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[a,b]});
 await expect.poll(()=>({x:input()?.x,guard:input()?.guard})).toEqual({x:-1,guard:true});
 await releasePointer(page,'left');
 await expect.poll(()=>({x:input()?.x,guard:input()?.guard})).toEqual({x:0,guard:true});
 await cdp.send('Input.dispatchTouchEvent',{type:'touchCancel',touchPoints:[]});
 await expect(guard).toHaveAttribute('aria-pressed','false');await expect.poll(()=>input()?.guard).toBe(false);
});


test('a skill can be cast while another finger holds movement',async({page})=>{
 const {left,cdp,seen,input}=await setup(page),skill=page.locator('#rift-signatures button').first();
 const a=point(await left.boundingBox(),1),b=point(await skill.boundingBox(),2);
 await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[a]});
 await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[a,b]});
 await expect.poll(()=>seen.some(value=>value.x===-1&&!!value.skill)).toBe(true);
 await releasePointer(page,await skill.getAttribute('data-action'));
 await expect(left).toHaveAttribute('aria-pressed','true');await expect.poll(()=>input()?.x).toBe(-1);
 await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
 await expect.poll(()=>input()?.x).toBe(0);
});


test('guard releases after its captured finger leaves the button',async({page})=>{
 const {cdp,input}=await setup(page),guard=page.locator('[data-bind=guard]');
 const finger=point(await guard.boundingBox(),1);
 await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[finger]});
 await expect.poll(()=>input()?.guard).toBe(true);
 await cdp.send('Input.dispatchTouchEvent',{type:'touchMove',touchPoints:[{...finger,x:10,y:10}]});
 await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
 await expect(guard).toHaveAttribute('aria-pressed','false');await expect.poll(()=>input()?.guard).toBe(false);
});
