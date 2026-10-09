import { createRequire } from 'node:module'
import { mkdir, writeFile } from 'node:fs/promises'
import assert from 'node:assert/strict'
const require=createRequire('/tmp/tocreate-preview-tools/package.json'),{chromium}=require('playwright')
const output=new URL('../preview-evidence/',import.meta.url).pathname
await mkdir(output,{recursive:true})
const browser=await chromium.launch({headless:true}),checks=[],errors=[],requests=[]
const origin='http://127.0.0.1:18084/'
async function setup(context){
 await context.route('https://api.lihe.chat/**',route=>route.abort())
 const page=await context.newPage();page.on('pageerror',e=>errors.push(e.message));page.on('request',r=>{if(/\/api\/|\/v1\//.test(r.url()))requests.push(r.url())});return page
}
async function ready(page,state='email'){
 await page.waitForFunction(s=>document.querySelector('.original-auth')?.dataset.step===s&&!document.querySelector('.email-auth-stage')?.inert&&!document.querySelector('.original-step-enter-active')&&!document.querySelector('.original-step-leave-active'),state)
 await page.waitForTimeout(350)
}
async function click(page,selector,touch=false){
 const r=await page.locator(selector).boundingBox();assert(r&&r.y>=0&&r.y+r.height<=page.viewportSize().height,selector+' visible without scroll')
 if(touch)await page.touchscreen.tap(r.x+r.width/2,r.y+r.height/2)
 else{await page.mouse.move(r.x+r.width/2,r.y+r.height/2,{steps:10});await page.mouse.click(r.x+r.width/2,r.y+r.height/2)}
}
async function layout(page,label){
 const result=await page.evaluate(()=>{
  const stage=document.querySelector('.email-auth-stage'),r=stage.getBoundingClientRect(),w=document.querySelector('.original-stage-window').getBoundingClientRect()
  return {width:innerWidth,scrollWidth:document.documentElement.scrollWidth,height:innerHeight,pageHeight:document.documentElement.scrollHeight,fields:document.querySelectorAll('form input').length,stageHeight:r.height,windowHeight:w.height,fits:r.bottom<=w.bottom+1,blur:getComputedStyle(stage).filter,focus:document.activeElement.id}
 })
 assert(result.scrollWidth<=result.width&&result.fits,label+JSON.stringify(result));assert(['none','blur(0px)'].includes(result.blur));checks.push({label,...result})
}
async function geometry(page){return page.evaluate(()=>Object.fromEntries(['.tc-auth-frame','.email-auth-brand','.email-auth-copy','.email-auth-input-shell','.email-auth-primary'].map(s=>{const e=document.querySelector(s),r=e.getBoundingClientRect(),c=getComputedStyle(e);return [s,{x:r.x,y:r.y,width:r.width,height:r.height,background:c.background,font:c.font,borderRadius:c.borderRadius}]})))}
try{
 for(const mobile of [false,true]){
  const name=mobile?'mobile':'desktop',viewport=mobile?{width:390,height:844}:{width:1280,height:900}
  const context=await browser.newContext({viewport,isMobile:mobile,hasTouch:mobile,deviceScaleFactor:1,recordVideo:{dir:output,size:viewport}}),page=await setup(context)
  await page.goto(origin);await page.waitForTimeout(500);await page.screenshot({path:output+name+'-home.png'})
  await click(page,'[data-action=start]',mobile);await ready(page);await layout(page,name+' email');await page.screenshot({path:output+name+'-email.png'})
  await page.evaluate(()=>{window.frame=document.querySelector('.tc-auth-frame');window.scrolls=[];window.addEventListener('scroll',()=>window.scrolls.push(scrollY))})
  await click(page,'[data-action=fill]',mobile);await page.waitForTimeout(350);await click(page,'[data-action=login]',mobile);await ready(page,'login');await layout(page,name+' login');await page.screenshot({path:output+name+'-login.png'})
  await click(page,'.email-auth-field-action',mobile);await page.waitForTimeout(450);await click(page,'.email-auth-field-action',mobile)
  await click(page,'[data-action=submit]',mobile);await page.waitForTimeout(250);await page.screenshot({path:output+name+'-loading.png'});await ready(page,'done');await layout(page,name+' done');await page.screenshot({path:output+name+'-done.png'});await page.waitForTimeout(600)
  await click(page,'[data-action=restart]',mobile);await ready(page);await click(page,'[data-action=register]',mobile);await ready(page,'register');await layout(page,name+' register');await page.screenshot({path:output+name+'-register.png'})
  await click(page,'[data-action=fill]',mobile);await page.waitForTimeout(500);await click(page,'[data-action=submit]',mobile);await ready(page,'verify');await layout(page,name+' verify');await page.screenshot({path:output+name+'-verify.png'})
  await click(page,'[data-action=fill]',mobile);await page.waitForTimeout(500);await click(page,'[data-action=submit]',mobile);await ready(page,'done');await page.waitForTimeout(600)
  await click(page,'[data-action=restart]',mobile);await ready(page);await click(page,'[data-action=login]',mobile);await ready(page,'login');await click(page,'[data-action=forgot]',mobile);await ready(page,'forgot');await layout(page,name+' forgot');await page.screenshot({path:output+name+'-forgot.png'})
  await click(page,'[data-action=submit]',mobile);await ready(page,'sent');await page.waitForTimeout(600)
  const continuity=await page.evaluate(()=>({frame:window.frame===document.querySelector('.tc-auth-frame'),scrolls:window.scrolls}));assert(continuity.frame&&continuity.scrolls.length===0,JSON.stringify(continuity));checks.push({label:name+' real interaction recording',...continuity})
  const video=page.video();await context.close();await video.saveAs(output+name+'-full-flow.webm')
 }
 const context=await browser.newContext({viewport:{width:1280,height:900}}),page=await setup(context)
 for(const width of [320,375,390,768,1280])for(const lang of ['zh','en']){
  await page.setViewportSize({width,height:844});await page.goto(origin+'?view=login&lang='+lang);await ready(page);await layout(page,lang+width+' email')
  const before=await geometry(page)
  await page.goto(origin+'?view=login&lang='+lang+'&still=1');await ready(page)
  assert.deepEqual(await geometry(page),before,'Motion must preserve original static appearance');checks.push({label:lang+width+' same static geometry and style with motion off',passed:true})
  await page.goto(origin+'?view=login&lang='+lang);await ready(page);await page.locator('[data-action=fill]').click();await page.locator('[data-action=login]').click();await ready(page,'login');await layout(page,lang+width+' login')
  assert.equal(await page.locator('form input').count(),1)
  await page.locator('[data-action=switch]').click();await ready(page,'register');await layout(page,lang+width+' register');assert.equal(await page.locator('form input').count(),2)
  await page.locator('[data-action=submit]').click();await ready(page,'verify');await layout(page,lang+width+' verify')
 }
 await page.goto(origin+'?view=login');await ready(page);await page.locator('[data-action=login]').click();await page.waitForTimeout(200)
 assert(await page.locator('#auth-message').evaluate(el=>el.classList.contains('is-error')))
 assert(await page.locator('#email-auth-email').evaluate(el=>el===document.activeElement))
 await page.keyboard.type('hello@example.com');await page.keyboard.press('Tab');await page.keyboard.press('Enter');await ready(page,'login')
 await page.keyboard.press('Tab');await page.keyboard.type('DemoOnly2026!');await page.keyboard.press('Enter');await ready(page,'done')
 assert(await page.locator('#email-auth-title').evaluate(el=>el===document.activeElement));checks.push({label:'keyboard email/login and validation',passed:true})
 await page.locator('[data-action=restart]').click();await ready(page);await page.locator('[data-action=register]').click();await ready(page,'register');await page.locator('[data-action=fill]').click()
 for(let i=0;i<4;i++){await page.locator('[data-action=switch]').click();await ready(page,i%2===0?'login':'register')}
 await page.locator('[data-action=submit]').evaluate(el=>{for(let i=0;i<10;i++)el.click()});await page.waitForTimeout(150);await page.locator('[data-action=cancel]').click();await page.waitForTimeout(1100)
 assert.equal(await page.locator('.original-auth').getAttribute('data-step'),'register');checks.push({label:'rapid operations, duplicate submit and cancel',passed:true})
 await page.locator('[data-action=submit]').click();await ready(page,'verify');await page.locator('#email-auth-code').fill('000000');await page.locator('[data-action=submit]').click();await page.waitForTimeout(200);assert(await page.locator('#auth-message').evaluate(el=>el.classList.contains('is-error')))
 await page.emulateMedia({reducedMotion:'reduce'});await page.locator('[data-action=fill]').click();await page.locator('[data-action=submit]').click();await page.waitForTimeout(150);assert.equal(await page.evaluate(()=>document.getAnimations().filter(a=>a.playState==='running').length),0);await ready(page,'done');checks.push({label:'code validation and reduced motion',passed:true})
 await page.locator('[data-action=restart]').click();await ready(page);await page.emulateMedia({reducedMotion:'no-preference'});await page.locator('[data-action=fill]').click();await page.locator('[data-action=login]').click();await ready(page,'login');await page.locator('[data-action=submit]').click();await page.locator('.tc-auth-home').click();await page.waitForTimeout(1100)
 assert.equal(await page.locator('.original-auth').count(),0);assert.equal(await page.evaluate(()=>document.getAnimations().filter(a=>a.playState==='running').length),0);checks.push({label:'unmount cleans simulated request and animations',passed:true})
 await page.setViewportSize({width:375,height:667});await page.goto(origin+'?view=login');await ready(page);await page.locator('[data-action=fill]').click();await page.locator('[data-action=login]').click();await ready(page,'login');await page.screenshot({path:output+'short-login.png'})
 const start=await page.locator('.tc-auth-frame').boundingBox();await page.locator('[data-action=switch]').click();await ready(page,'register');await page.screenshot({path:output+'short-register.png'});const end=await page.locator('.tc-auth-frame').boundingBox();assert(Math.abs(start.height-end.height)<1);checks.push({label:'original email-first flow has no third field or height jump',start:start.height,end:end.height})
 assert.equal(errors.length,0);assert.equal(requests.length,0);await context.close()
}finally{await writeFile(output+'verification.json',JSON.stringify({checks,errors,apiRequests:requests},null,2));await browser.close()}
