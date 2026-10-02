const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true,viewport:{width:390,height:844}});

test('combat buttons suppress context menus while document content stays normal',async({page})=>{
 await page.goto('/abyss/rift?practice=skills');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 const controls=await page.locator('[data-action]').evaluateAll(nodes=>nodes.filter(node=>node.getClientRects().length).map(node=>{
  const event=new MouseEvent('contextmenu',{bubbles:true,cancelable:true});(node.querySelector('span')||node).dispatchEvent(event);
  const style=getComputedStyle(node);return {action:node.dataset.action,prevented:event.defaultPrevented,select:style.userSelect,touch:style.touchAction};
 }));
 expect(controls.length).toBeGreaterThan(8);
 expect(controls.filter(control=>!control.prevented||control.select!=='none'||control.touch!=='none')).toEqual([]);
 const outside=await page.locator('#rift-practice-instructions').evaluate(node=>{
  const event=new MouseEvent('contextmenu',{bubbles:true,cancelable:true});node.dispatchEvent(event);
  const style=getComputedStyle(node);return {prevented:event.defaultPrevented,select:style.userSelect,touch:style.touchAction};
 });
 expect(outside.prevented).toBe(false);expect(outside.select).not.toBe('none');expect(outside.touch).toBe('auto');
 await page.locator('#rift-pause').tap();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
 expect(await page.locator('[data-bind=attack]').evaluate(node=>{const event=new MouseEvent('contextmenu',{bubbles:true,cancelable:true});node.dispatchEvent(event);return event.defaultPrevented;})).toBe(false);
});
