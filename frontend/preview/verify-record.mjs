import { createRequire } from 'node:module'
import { mkdir, writeFile } from 'node:fs/promises'
import assert from 'node:assert/strict'
const require = createRequire('/tmp/tocreate-preview-tools/package.json')
const { chromium } = require('playwright')
const output = new URL('../preview-evidence/', import.meta.url).pathname
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true })
const checks = [], errors = [], requests = [], destinations = []
const origin = 'http://127.0.0.1:18084/'
async function setup(context) {
  // Observe real link activation without accessing the live authentication service.
  await context.route('https://api.lihe.chat/**', async route => {
    destinations.push(route.request().url())
    await route.abort('aborted')
  })
  const page = await context.newPage()
  page.on('pageerror', e => errors.push(e.message))
  page.on('request', r => { if (/\/api\/|\/v1\//.test(r.url())) requests.push(r.url()) })
  return page
}
async function layout(page, label) {
  const result = await page.evaluate(() => {
    const links = [...document.querySelectorAll('.entry-link')].map(el => {
      const r = el.getBoundingClientRect()
      return { x:r.x,y:r.y,width:r.width,height:r.height,href:el.href,label:el.innerText }
    })
    return { width:innerWidth, scrollWidth:document.documentElement.scrollWidth, y:scrollY, links, text:document.querySelector('main').innerText, card:!!document.querySelector('.mg-demo') }
  })
  assert(result.scrollWidth <= result.width, label+' horizontal overflow')
  assert(!result.card && !/Claude|GPT|Gemini|密钥|API key|multiple models/i.test(result.text), label+' removed content')
  assert(result.links.every(r=>r.height>=44&&r.x>=0&&r.x+r.width<=result.width&&r.href==='https://api.lihe.chat/login'),label+' accessible entry links')
  checks.push({label,...result})
}
async function point(page, selector) {
  const r=await page.locator(selector).boundingBox()
  assert(r&&r.y>=0&&r.y+r.height<=page.viewportSize().height,'Visible without scrolling: '+selector)
  return {x:r.x+r.width/2,y:r.y+r.height/2}
}
try {
  for(const mobile of [false,true]) {
    const name=mobile?'mobile':'desktop',viewport=mobile?{width:390,height:844}:{width:1280,height:900}
    const context=await browser.newContext({viewport,isMobile:mobile,hasTouch:mobile,deviceScaleFactor:1,recordVideo:{dir:output,size:viewport}})
    const page=await setup(context)
    await page.goto(origin);await page.evaluate(()=>document.fonts.ready);await page.waitForTimeout(800)
    await page.evaluate(()=>{window.scrollSamples=[];window.addEventListener('scroll',()=>window.scrollSamples.push(scrollY));window.startRect=document.querySelector('.preview-hero').getBoundingClientRect().toJSON()})
    await layout(page,name+' initial');await page.screenshot({path:output+name+'-initial.png'})
    const p=await point(page,'[data-action=start]')
    if(mobile){await page.touchscreen.tap(p.x,p.y)}
    else {
      await page.mouse.move(40,400);await page.mouse.move(p.x,p.y,{steps:24});await page.waitForTimeout(900)
      await page.screenshot({path:output+name+'-hover.png'})
      await page.mouse.down();await page.waitForTimeout(220);await page.screenshot({path:output+name+'-pressed.png'});await page.mouse.up()
    }
    await page.waitForTimeout(650)
    assert.equal(page.url(),origin,'Aborted outbound navigation must preserve preview')
    const login=await point(page,'[data-action=login]')
    if(mobile)await page.touchscreen.tap(login.x,login.y)
    else {await page.mouse.move(login.x,login.y,{steps:24});await page.waitForTimeout(700);await page.mouse.click(login.x,login.y)}
    await page.waitForTimeout(650)
    const language=await point(page,'.preview-language')
    if(mobile)await page.touchscreen.tap(language.x,language.y)
    else {await page.mouse.move(language.x,language.y,{steps:20});await page.mouse.click(language.x,language.y)}
    await page.waitForTimeout(700);await layout(page,name+' English');await page.screenshot({path:output+name+'-english.png'})
    await page.keyboard.press('Tab');await page.keyboard.press('Tab');await page.waitForTimeout(900)
    await page.screenshot({path:output+name+'-focus.png'})
    await page.keyboard.press('Enter');await page.waitForTimeout(600)
    const continuity=await page.evaluate(()=>({scrolls:window.scrollSamples,heroShift:Math.abs(document.querySelector('.preview-hero').getBoundingClientRect().y-window.startRect.y)}))
    assert(continuity.scrolls.length===0&&continuity.heroShift<1)
    checks.push({label:name+' interaction stability',...continuity})
    const video=page.video();await context.close();await video.saveAs(output+name+'-full-flow.webm')
  }
  const context=await browser.newContext({viewport:{width:1280,height:900}})
  const page=await setup(context)
  for(const width of [320,360,390,768,1280])for(const lang of ['zh','en']){
    await page.setViewportSize({width,height:844});await page.goto(origin+'?lang='+lang)
    await layout(page,lang+' '+width)
  }
  await page.setViewportSize({width:390,height:500});await page.goto(origin)
  await layout(page,'short mobile viewport')
  await point(page,'[data-action=start]')
  await page.goto(origin)
  for(let i=0;i<4;i++)await page.keyboard.press('Tab')
  assert(await page.locator('[data-action=start]').evaluate(el=>el===document.activeElement),'Keyboard order reaches primary link')
  assert(await page.locator('[data-action=start]').evaluate(el=>getComputedStyle(el).outlineStyle!=='none'),'Visible keyboard focus')
  const before=destinations.length
  await page.keyboard.press('Enter');await page.waitForTimeout(200)
  assert.equal(destinations.length,before+1,'Enter navigates exactly once')
  checks.push({label:'keyboard focus and immediate navigation',passed:true})
  await page.emulateMedia({reducedMotion:'reduce'})
  await page.locator('[data-action=start]').hover()
  const reduced=await page.locator('[data-action=start]').evaluate(el=>({transition:getComputedStyle(el).transitionDuration,iconTransform:getComputedStyle(el.querySelector('svg')).transform}))
  assert.equal(reduced.transition,'0s');assert.equal(reduced.iconTransform,'none')
  checks.push({label:'reduced motion',...reduced})
  // Repeated pointer interactions cannot create request timers or dynamic states.
  for(let i=0;i<6;i++)await page.locator('[data-action=start]').click({noWaitAfter:true})
  await page.waitForTimeout(300);await layout(page,'repeated activation')
  assert.equal(await page.evaluate(()=>document.getAnimations().filter(a=>a.playState==='running').length),0)
  assert(destinations.length>=6&&destinations.every(url=>url==='https://api.lihe.chat/login'))
  checks.push({label:'entry destinations',count:destinations.length,target:destinations[0]})
  assert.equal(errors.length,0);assert.equal(requests.length,0)
  await context.close()
} finally {
  await writeFile(output+'verification.json',JSON.stringify({checks,errors,apiRequests:requests,navigationNote:'Outbound login navigation intercepted and aborted by the test only; product links navigate immediately.'},null,2))
  await browser.close()
}
