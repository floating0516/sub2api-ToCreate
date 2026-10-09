import { createRequire } from 'node:module'
import { mkdir, writeFile } from 'node:fs/promises'
import assert from 'node:assert/strict'
const require = createRequire('/tmp/tocreate-preview-tools/package.json')
const { chromium } = require('playwright')
const output = new URL('../preview-evidence/', import.meta.url).pathname
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true })
const checks = [], errors = [], requests = []
const wait = ms => new Promise(resolve => setTimeout(resolve, ms))
async function rawClick(page, selector, mobile = false) {
  const r = await page.locator(selector).boundingBox()
  assert(r && r.y >= 0 && r.y + r.height <= page.viewportSize().height, 'Click target must be visible without auto-scroll: ' + selector)
  if (mobile) await page.touchscreen.tap(r.x + r.width / 2, r.y + r.height / 2)
  else { await page.mouse.move(r.x + r.width / 2, r.y + r.height / 2, { steps: 12 }); await page.mouse.click(r.x + r.width / 2, r.y + r.height / 2) }
}
async function settled(page, state) {
  await page.waitForFunction(value => document.querySelector('.mg-demo')?.dataset.state === value && !document.querySelector('.mg-content')?.inert, state)
  await wait(500)
}
async function layout(page, label) {
  const r = await page.evaluate(() => {
    const shape = document.querySelector('.mg-shape').getBoundingClientRect(), inner = document.querySelector('.mg-content').getBoundingClientRect(), stage = document.querySelector('.mg-stage').getBoundingClientRect(), mark = document.querySelector('.mg-shared-mark').getBoundingClientRect(), slot = document.querySelector('.mg-mark-slot').getBoundingClientRect()
    return { width:innerWidth, scrollWidth:document.documentElement.scrollWidth, stageHeight:stage.height, shapeHeight:shape.height, contentFits:inner.top >= shape.top && inner.bottom <= shape.bottom && inner.left >= shape.left && inner.right <= shape.right, stageFits:shape.top >= stage.top && shape.bottom <= stage.bottom, markOffset:Math.max(Math.abs(mark.left-slot.left),Math.abs(mark.top-slot.top)), blur:getComputedStyle(document.querySelector('.mg-content')).filter, y:scrollY }
  })
  assert(r.scrollWidth <= r.width, label+' overflow')
  assert(r.contentFits && r.stageFits, label+' clipping '+JSON.stringify(r))
  assert(r.markOffset < 3, label+' marker misaligned '+JSON.stringify(r))
  assert(['none','blur(0px)'].includes(r.blur))
  checks.push({ label, ...r })
}
try {
  for (const mobile of [false,true]) {
    const name = mobile ? 'mobile' : 'desktop', viewport = mobile ? {width:390,height:844} : {width:1280,height:900}
    const context = await browser.newContext({ viewport, isMobile:mobile, hasTouch:mobile, deviceScaleFactor:1, recordVideo:{dir:output,size:viewport} })
    const page = await context.newPage()
    page.on('pageerror',e=>errors.push(e.message));page.on('request',r=>{if(/\/api\/|\/v1\//.test(r.url()))requests.push(r.url())})
    await page.goto('http://127.0.0.1:18084/')
    await page.locator('.mg-shape').waitFor(); await page.evaluate(()=>document.fonts.ready);await wait(900)
    await page.evaluate(()=>{window.originalMark=document.querySelector('.mg-shared-mark');window.originalShape=document.querySelector('.mg-shape');window.scrollSamples=[];window.addEventListener('scroll',()=>window.scrollSamples.push(scrollY));window.following=document.querySelector('.preview-following').getBoundingClientRect().top+scrollY})
    await layout(page,name+' key');await page.screenshot({path:output+name+'-key.png'})
    await rawClick(page,'[data-action=choose]',mobile);await settled(page,'select');await wait(800)
    for(const index of [1,2,0,2]){await rawClick(page,`[data-model="${index}"]`,mobile);await wait(650)}
    await layout(page,name+' select');await page.screenshot({path:output+name+'-select.png'})
    await rawClick(page,'[data-action=request]',mobile);await settled(page,'loading');await page.screenshot({path:output+name+'-loading.png'})
    await settled(page,'response');await layout(page,name+' response');await page.screenshot({path:output+name+'-response.png'});await wait(2100)
    await rawClick(page,'[data-action=again]',mobile);await settled(page,'select');await rawClick(page,'[data-model="1"]',mobile);await wait(600)
    await rawClick(page,'[data-action=request]',mobile);await settled(page,'response');await wait(1600)
    await rawClick(page,'[data-action=done]',mobile);await settled(page,'done');await layout(page,name+' done');await page.screenshot({path:output+name+'-done.png'});await wait(1100)
    await rawClick(page,'[data-action=restart]',mobile);await settled(page,'key');await wait(1000)
    const continuity=await page.evaluate(()=>({sameMark:window.originalMark===document.querySelector('.mg-shared-mark'),sameShape:window.originalShape===document.querySelector('.mg-shape'),scrolls:window.scrollSamples,followingDelta:Math.abs(window.following-(document.querySelector('.preview-following').getBoundingClientRect().top+scrollY))}))
    assert(continuity.sameMark&&continuity.sameShape&&continuity.followingDelta<1)
    assert(continuity.scrolls.every(y=>y===0), 'Unrequested scroll during complete recording')
    checks.push({label:name+' real-time recorded flow',...continuity})
    const video=page.video();await context.close();await video.saveAs(output+name+'-full-flow.webm')
  }
  const context=await browser.newContext({viewport:{width:390,height:600},hasTouch:true,isMobile:true})
  const page=await context.newPage();page.on('pageerror',e=>errors.push(e.message))
  for (const width of [320,360,390,768]) for(const lang of ['zh','en']) {
    await page.setViewportSize({width,height:844});await page.goto('http://127.0.0.1:18084/?lang='+lang);await wait(400)
    await layout(page,lang+width+' key')
    await page.locator('[data-action=choose]').click();await settled(page,'select');await layout(page,lang+width+' select')
    for(const n of [2,0,1,2,0])await page.locator(`[data-model="${n}"]`).click()
    await page.locator('[data-action=request]').click();await settled(page,'response');await layout(page,lang+width+' response')
    await page.locator('[data-action=done]').click();await settled(page,'done');await layout(page,lang+width+' done')
  }
  // Reproduce the previous offscreen-radio arrow key path without auto-scrolling locators.
  await page.setViewportSize({width:390,height:500});await page.goto('http://127.0.0.1:18084/')
  await page.locator('[data-action=choose]').evaluate(el=>el.click());await settled(page,'select')
  await page.locator('[data-model="0"]').evaluate(el=>el.focus({preventScroll:true}));await page.evaluate(()=>scrollTo(0,570));await wait(100)
  const before=await page.evaluate(()=>scrollY);await page.keyboard.press('ArrowRight');await wait(500);const after=await page.evaluate(()=>scrollY)
  assert.equal(after,before);checks.push({label:'offscreen radio arrow key scroll regression',before,after})
  // Also verify an ordinary visible keyboard flow, including focus after async completion.
  await page.setViewportSize({width:1280,height:900});await page.goto('http://127.0.0.1:18084/')
  for(let i=0;i<10;i++){await page.keyboard.press('Tab');if(await page.locator('[data-action=choose]').evaluate(el=>el===document.activeElement))break}
  await page.keyboard.press('Enter');await settled(page,'select');await page.keyboard.press('End');await page.keyboard.press('Tab');await page.keyboard.press('Enter');await settled(page,'response')
  assert.equal(await page.evaluate(()=>document.activeElement.tagName),'H3')
  await page.keyboard.press('Tab');await page.keyboard.press('Tab');await page.keyboard.press('Enter');await settled(page,'done');await page.keyboard.press('Enter');await settled(page,'key');checks.push({label:'full keyboard flow',passed:true})
  await page.emulateMedia({reducedMotion:'reduce'});await page.locator('[data-action=choose]').click();await settled(page,'select');await page.locator('[data-action=request]').click();await settled(page,'loading')
  assert.equal(await page.evaluate(()=>document.getAnimations().filter(a=>a.playState==='running').length),0);await settled(page,'response');await layout(page,'reduced motion response')
  await page.locator('[data-action=reset]').click();await settled(page,'key');await page.emulateMedia({reducedMotion:'no-preference'})
  await page.locator('[data-action=choose]').click();await settled(page,'select');await page.locator('[data-action=request]').evaluate(el=>{for(let i=0;i<12;i++)el.click()});await wait(100);await page.locator('[data-action=reset]').click();await settled(page,'key');await wait(1300);assert.equal(await page.locator('.mg-demo').getAttribute('data-state'),'key');checks.push({label:'repeated submission and stale callback reset',passed:true})
  assert.equal(errors.length,0);assert.equal(requests.length,0)
  await context.close()
} finally {
  await writeFile(output+'verification.json',JSON.stringify({checks,errors,apiRequests:requests},null,2));await browser.close()
}
