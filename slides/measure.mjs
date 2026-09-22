import { chromium } from 'playwright-chromium'
import { createServer } from 'node:http'
import { readFile } from 'node:fs/promises'
import { extname, join } from 'node:path'
const dir = process.argv[2]
const TYPES = { '.html':'text/html','.js':'text/javascript','.css':'text/css','.png':'image/png','.json':'application/json','.woff2':'font/woff2','.svg':'image/svg+xml' }
const srv = createServer(async (req, res) => {
  const p = decodeURIComponent(req.url.split('?')[0])
  try { const b = await readFile(join(dir, p === '/' ? 'index.html' : p)); res.writeHead(200,{'content-type':TYPES[extname(p)]||'application/octet-stream'}); res.end(b) }
  catch { const b = await readFile(join(dir,'index.html')); res.writeHead(200,{'content-type':'text/html'}); res.end(b) }
})
await new Promise(r => srv.listen(8808, r))
const b = await chromium.launch()
const p = await b.newPage({ viewport: { width: 1920, height: 1080 } })
for (const n of [1, 2, 5, 6, 12]) {
  await p.goto(`http://localhost:8808/${n}`, { waitUntil: 'networkidle' })
  await p.waitForTimeout(2500)
  const r = await p.evaluate(() => {
    const rows = [...document.querySelectorAll('.slidev-layout div')].filter(e => getComputedStyle(e).display === 'flex' && e.children.length === 2)
    if (!rows.length) return null
    const row = rows[0]
    const marker = row.children[0], text = row.children[1]
    const box = row.parentElement
    return {
      containerLeft: +box.getBoundingClientRect().left.toFixed(1),
      containerWidth: +box.getBoundingClientRect().width.toFixed(1),
      markerLeft: +marker.getBoundingClientRect().left.toFixed(1),
      markerWidth: +marker.getBoundingClientRect().width.toFixed(1),
      textLeft: +text.getBoundingClientRect().left.toFixed(1),
    }
  })
  console.log('slide', String(n).padStart(2), JSON.stringify(r))
}
await b.close(); srv.close()
