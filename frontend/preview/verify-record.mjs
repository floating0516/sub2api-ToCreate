import { createRequire } from 'node:module'
import { mkdir, writeFile } from 'node:fs/promises'
import assert from 'node:assert/strict'
const require = createRequire('/tmp/tocreate-preview-tools/package.json')
const { chromium } = require('playwright')
const output = new URL('../preview-evidence/', import.meta.url).pathname
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true })
const checks=[], errors=[], requests=[]
const origin='http://127.0.0.1:18084/'
async function setup(context){
  await context.route('https://api.lihe.chat/**',route=>route.abort())
  const page=await context.newPage()
  page.on('pageerror',e=>errors.push(e.message))
  page.on('request',r=>{if(/\/api\/|\/v1\//.test(r.url()))requests.push(r.url())})
  return page
}
async function ready(page,step='form'){
  await page.waitForFunction(value=>document.querySelector('.auth-preview')?.dataset.step===value&&!document.querySelector('#auth-panel')?.inert&&!document.querySelector('.auth-content-enter-active')&&!document.querySelector('.auth-content-leave-active'),step)
  await page.waitForTimeout(450)
}
async function click(page,selector,mobile=false){
  const r=await page.locator(selector).boundingBox()
  assert(r&&r.y>=0&&r.y+r.height<=page.viewportSize().height,'Visible target without auto-scroll: '+selector)
  if(mobile)await page.touchscreen.tap(r.x+r.width/2,r.y+r.height/2)
  else{await page.mouse.move(r.x+r.width/2,r.y+r.height/2,{steps:10});await page.mouse.click(r.x+r.width/2,r.y+r.height/2)}
}
async function layout(page,label){
  const value=await page.evaluate(()=>{
    const shape=document.querySelector('.auth-shape'),inside=document.querySelector('.auth-inside'),r=shape.getBoundingClientRect(),i=inside.getBoundingClientRect()
    const panel=document.querySelector('#auth-panel')
    return {width:innerWidth,scrollWidth:document.documentElement.scrollWidth,y:scrollY,height:r.height,contentHeight:i.height,fits:i.bottom<=r.bottom+1,blur:getComputedStyle(panel).filter,panels:document.querySelectorAll('#auth-panel').length,focus:document.activeElement.id,fieldWidths:[...document.querySelectorAll('.auth-field input')].map(el=>el.getBoundingClientRect().width)}
  })
  assert(value.scrollWidth<=value.width&&value.fits&&value.panels===1,label+JSON.stringify(value))
  assert(value.blur==='none'||value.blur==='blur(0px)')
  assert(value.fieldWidths.every(w=>w>=180))
  checks.push({label,...value})
}
try{
  for(const mobile of [false,true]){
    const name=mobile?'mobile':'desktop',viewport=mobile?{width:390,height:844}:{width:1280,height:900}
    const context=await browser.newContext({viewport,isMobile:mobile,hasTouch:mobile,deviceScaleFactor:1,recordVideo:{dir:output,size:viewport}})
    const page=await setup(context)
    await page.goto(origin);await page.waitForTimeout(700)
    await click(page,'[data-action=start]',mobile);await ready(page)
    await page.evaluate(()=>{window.initialShape=document.querySelector('.auth-shape');window.initialMark=document.querySelector('.auth-mark svg');window.scrolls=[];window.addEventListener('scroll',()=>window.scrolls.push(scrollY))})
    await layout(page,name+' login');await page.screenshot({path:output+name+'-login.png'})
    await click(page,'[data-action=fill]',mobile);await page.waitForTimeout(450)
    await click(page,'.auth-reveal',mobile);await page.waitForTimeout(500);await click(page,'.auth-reveal',mobile)
    await click(page,'[data-action=submit]',mobile);await page.waitForTimeout(300);await page.screenshot({path:output+name+'-loading.png'})
    await ready(page,'done');await layout(page,name+' login done');await page.screenshot({path:output+name+'-done.png'});await page.waitForTimeout(700)
    await click(page,'[data-action=restart]',mobile);await ready(page)
    await click(page,'#register-tab',mobile);await ready(page);await layout(page,name+' register');await page.screenshot({path:output+name+'-register.png'})
    await click(page,'[data-action=fill]',mobile);await page.waitForTimeout(500);await click(page,'[data-action=submit]',mobile)
    await ready(page,'verify');await layout(page,name+' verify');await page.screenshot({path:output+name+'-verify.png'})
    await click(page,'[data-action=fill]',mobile);await page.waitForTimeout(700);await click(page,'[data-action=submit]',mobile)
    await ready(page,'done');await page.waitForTimeout(700)
    await click(page,'[data-action=restart]',mobile);await ready(page);await click(page,'#login-tab',mobile);await ready(page)
    await click(page,'[data-action=forgot]',mobile);await ready(page,'forgot');await layout(page,name+' forgot');await page.screenshot({path:output+name+'-forgot.png'})
    await click(page,'[data-action=fill]',mobile);await click(page,'[data-action=submit]',mobile);await ready(page,'sent');await page.waitForTimeout(850)
    const continuity=await page.evaluate(()=>({sameShape:window.initialShape===document.querySelector('.auth-shape'),sameMark:window.initialMark===document.querySelector('.auth-mark svg'),scrolls:window.scrolls}))
    assert(continuity.sameShape&&continuity.sameMark&&continuity.scrolls.length===0,JSON.stringify(continuity));checks.push({label:name+' continuity',...continuity})
    const video=page.video();await context.close();await video.saveAs(output+name+'-full-flow.webm')
  }
  const context=await browser.newContext({viewport:{width:1280,height:900}}),page=await setup(context)
  for(const width of [320,360,390,768,1280])for(const lang of ['zh','en']){
    await page.setViewportSize({width,height:900});await page.goto(origin+'?view=login&lang='+lang);await ready(page);await layout(page,lang+width+' login')
    await page.locator('#register-tab').click();await ready(page);await layout(page,lang+width+' register')
    await page.locator('[data-action=submit]').click();await page.waitForTimeout(500)
    assert(await page.locator('#auth-email').evaluate(el=>el===document.activeElement));await layout(page,lang+width+' errors')
    await page.locator('[data-action=fill]').click();await page.locator('[data-action=submit]').click();await ready(page,'verify');await layout(page,lang+width+' code')
  }
  await page.setViewportSize({width:1280,height:900});await page.goto(origin+'?view=login');await ready(page)
  await page.locator('#login-tab').focus();await page.keyboard.press('ArrowRight');await ready(page)
  assert.equal(await page.locator('#register-tab').getAttribute('aria-selected'),'true')
  assert(await page.locator('#register-tab').evaluate(el=>el===document.activeElement))
  await page.keyboard.press('Tab');await page.keyboard.type('hello@example.com');await page.keyboard.press('Tab');await page.keyboard.type('DemoOnly2026!');await page.keyboard.press('Tab');await page.keyboard.press('Tab');await page.keyboard.type('DemoOnly2026!');await page.keyboard.press('Enter');await ready(page,'verify')
  assert(await page.locator('#auth-title').evaluate(el=>el===document.activeElement))
  await page.keyboard.press('Tab');await page.keyboard.type('000000');await page.keyboard.press('Enter');await page.waitForTimeout(450)
  assert(await page.locator('#code-error').isVisible())
  await page.locator('#auth-code').fill('123456');await page.keyboard.press('Enter');await ready(page,'done')
  await page.keyboard.press('Tab');await page.keyboard.press('Enter');await ready(page);checks.push({label:'keyboard registration and code validation',passed:true})
  // Rapid toggles, duplicate submission and cancellation must never revive stale results.
  await page.locator('#login-tab').click();await ready(page)
  await page.locator('#register-tab').evaluate(el=>{for(let i=0;i<12;i++)el.click()});await ready(page)
  await page.locator('[data-action=fill]').click();await page.locator('[data-action=submit]').evaluate(el=>{for(let i=0;i<10;i++)el.click()});await page.waitForTimeout(180)
  await page.locator('[data-action=cancel]').click();await page.waitForTimeout(1300)
  assert.equal(await page.locator('.auth-preview').getAttribute('data-step'),'form');assert.equal(await page.locator('.auth-preview').getAttribute('data-busy'),'false')
  await page.locator('[data-action=submit]').click();await page.waitForTimeout(100);await page.locator('#login-tab').click();await ready(page);await page.waitForTimeout(1200)
  assert.equal(await page.locator('.auth-preview').getAttribute('data-step'),'form');checks.push({label:'rapid switching and stale request cancellation',passed:true})
  await page.emulateMedia({reducedMotion:'reduce'});await page.locator('[data-action=fill]').click();await page.locator('[data-action=submit]').click();await page.waitForTimeout(200)
  assert.equal(await page.evaluate(()=>document.getAnimations().filter(a=>a.playState==='running').length),0)
  await ready(page,'done');await layout(page,'reduced motion success');await page.screenshot({path:output+'reduced-motion.png'})
  await page.locator('[data-action=restart]').click();await ready(page);await page.emulateMedia({reducedMotion:'no-preference'})
  await page.locator('[data-action=fill]').click();await page.locator('[data-action=submit]').click();await page.waitForTimeout(100);await page.locator('.preview-brand').click();await page.waitForTimeout(1300)
  assert.equal(await page.locator('.auth-preview').count(),0);assert.equal(await page.evaluate(()=>document.getAnimations().filter(a=>a.playState==='running').length),0)
  await page.goBack();await ready(page);await page.reload();await ready(page);checks.push({label:'unmount cleanup, browser back and reload',passed:true})
  await page.setViewportSize({width:390,height:500});await page.locator('#register-tab').click();await ready(page);await layout(page,'short screen registration')
  const p=await page.locator('#auth-confirm').boundingBox();assert(p.height>=44)
  assert.equal(errors.length,0);assert.equal(requests.length,0)
  await context.close()
}finally{await writeFile(output+'verification.json',JSON.stringify({checks,errors,apiRequests:requests},null,2));await browser.close()}
