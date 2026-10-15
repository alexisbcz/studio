// The canvas: renders the HTML the engine produced, maps elements back to
// the calls of the source (data-studio-src) and reports selection, hover
// and drops to the studio (studio.js), which owns the source.

const root = document.getElementById("root")
const ATTR = "data-studio-src"
const SNIPPET = "application/x-studio-snippet"
let names = {} // id -> "button.Button"
let mode = "select"
let selected = null // id
let hovered = null
let drop = null // { id, where, el }

const overlay = document.createElement("div")
overlay.id = "studio-overlay"
document.body.append(overlay)
document.documentElement.dataset.mode = mode

const post = (msg) => parent.postMessage(msg, "*")

function elementsOf(id) {
  return id == null ? [] : [...root.querySelectorAll(`[${ATTR}="${id}"]`)]
}

// target returns the element of the innermost instrumented call at an
// event's target.
function target(el) {
  while (el && el !== root && !(el instanceof Element && el.hasAttribute(ATTR))) el = el.parentElement
  return el && el !== root ? el : null
}

// --- rendering ---

function render(html) {
  const scroll = [document.scrollingElement.scrollLeft, document.scrollingElement.scrollTop]
  root.innerHTML = html
  document.scrollingElement.scrollTo(...scroll)
  post({ type: "tree", tree: tree(root, false) })
}

// tree lists the instrumented elements, nested as in the DOM; elements of
// templates (closed dialogs, popovers) are listed too, as hidden.
function tree(node, hidden) {
  const out = []
  const walk = (parent, list, hide) => {
    for (const el of parent.children) {
      if (el.tagName === "TEMPLATE") {
        walk(el.content, list, true)
        continue
      }
      if (el.hasAttribute(ATTR)) {
        const item = { id: Number(el.getAttribute(ATTR)), tag: el.tagName.toLowerCase(), text: label(el), hidden: hide, children: [] }
        list.push(item)
        walk(el, item.children, hide)
      } else {
        walk(el, list, hide)
      }
    }
  }
  walk(node, out, hidden)
  return out
}

function label(el) {
  let text = ""
  for (const n of el.childNodes) if (n.nodeType === 3) text += n.textContent
  text = text.trim().replace(/\s+/g, " ")
  if (!text && el.getAttribute("aria-label")) text = el.getAttribute("aria-label")
  if (!text && el.placeholder) text = el.placeholder
  return text.length > 40 ? text.slice(0, 39) + "…" : text
}

// --- overlay ---

function boxes() {
  overlay.replaceChildren()
  const add = (rect, cls) => {
    const d = document.createElement("div")
    d.className = "box " + cls
    Object.assign(d.style, { left: rect.left + "px", top: rect.top + "px", width: rect.width + "px", height: rect.height + "px" })
    overlay.append(d)
    return d
  }
  if (hovered != null && hovered !== selected && mode === "select") {
    for (const el of elementsOf(hovered)) add(el.getBoundingClientRect(), "hover")
  }
  const sel = elementsOf(selected)
  sel.forEach((el, i) => {
    const r = el.getBoundingClientRect()
    if (!r.width && !r.height) return
    add(r, "selected")
    if (i === 0) {
      const l = document.createElement("div")
      l.className = "label"
      l.textContent = names[selected] ?? ""
      Object.assign(l.style, { left: r.left + "px", top: Math.max(r.top, 16) + "px" })
      overlay.append(l)
    }
  })
  if (drop) {
    const r = drop.el.getBoundingClientRect()
    if (drop.where === "inside") add(r, "drop-inside")
    else {
      const line = document.createElement("div")
      line.className = "drop-line"
      Object.assign(line.style, { left: r.left + "px", width: r.width + "px", top: (drop.where === "before" ? r.top - 2 : r.bottom - 1) + "px" })
      overlay.append(line)
    }
  }
}

let frame = 0
function schedule() {
  if (!frame) frame = requestAnimationFrame(() => { frame = 0; boxes() })
}
addEventListener("scroll", schedule, true)
addEventListener("resize", schedule)
new MutationObserver(schedule).observe(root, { subtree: true, childList: true, attributes: true })
setInterval(schedule, 500) // animations

// --- selection ---

function blocking(event) {
  if (mode !== "select") return false
  return event.target instanceof Node && root.contains(event.target)
}

for (const type of ["pointerdown", "mousedown", "pointerup", "mouseup", "dblclick", "auxclick", "contextmenu"]) {
  addEventListener(type, (e) => { if (blocking(e)) { e.preventDefault(); e.stopPropagation() } }, true)
}

addEventListener("click", (e) => {
  // Links never navigate the canvas away.
  const a = e.target instanceof Element && e.target.closest("a[href]")
  if (a && !a.getAttribute("href").startsWith("#")) e.preventDefault()
  if (!blocking(e)) return
  e.preventDefault()
  e.stopPropagation()
  const el = target(e.target)
  const id = el ? Number(el.getAttribute(ATTR)) : null
  post({ type: "select", id })
}, true)

addEventListener("submit", (e) => e.preventDefault(), true)

addEventListener("mousemove", (e) => {
  if (mode !== "select") return
  const el = target(e.target)
  const id = el ? Number(el.getAttribute(ATTR)) : null
  if (id !== hovered) {
    hovered = id
    post({ type: "hover", id })
    schedule()
  }
})
addEventListener("mouseleave", () => { hovered = null; post({ type: "hover", id: null }); schedule() })

// Keyboard shortcuts go to the studio while the canvas has focus.
addEventListener("keydown", (e) => {
  const editable = e.target instanceof Element && e.target.closest("input, textarea, select, [contenteditable]")
  if (mode === "select" && !editable) {
    const keys = ["Delete", "Backspace", "Escape", "ArrowUp", "ArrowDown"]
    if (keys.includes(e.key) || ((e.metaKey || e.ctrlKey) && ["d", "k", "z", "s"].includes(e.key.toLowerCase()))) {
      e.preventDefault()
      post({ type: "key", key: e.key, meta: e.metaKey || e.ctrlKey, shift: e.shiftKey, alt: e.altKey })
    }
  } else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
    e.preventDefault()
    post({ type: "key", key: e.key, meta: true })
  }
}, true)

// --- drag and drop from the palette ---

function dropAt(e) {
  const el = target(e.target)
  if (!el) {
    // Empty space: the page's first element.
    const first = root.querySelector(`[${ATTR}]`)
    return first ? { id: Number(first.getAttribute(ATTR)), where: "inside", el: first } : null
  }
  const r = el.getBoundingClientRect()
  const y = (e.clientY - r.top) / Math.max(r.height, 1)
  const where = y < 0.25 ? "before" : y > 0.75 ? "after" : "inside"
  return { id: Number(el.getAttribute(ATTR)), where, el }
}

addEventListener("dragover", (e) => {
  if (!e.dataTransfer.types.includes(SNIPPET)) return
  e.preventDefault()
  e.dataTransfer.dropEffect = "copy"
  drop = dropAt(e)
  schedule()
})
addEventListener("dragleave", (e) => {
  if (e.relatedTarget === null) { drop = null; schedule() }
})
addEventListener("drop", (e) => {
  if (!e.dataTransfer.types.includes(SNIPPET)) return
  e.preventDefault()
  const d = dropAt(e)
  drop = null
  schedule()
  if (d) post({ type: "drop", snippet: Number(e.dataTransfer.getData(SNIPPET)), id: d.id, where: d.where })
})

// --- messages from the studio ---

addEventListener("message", (e) => {
  if (e.source !== parent) return
  const m = e.data
  switch (m.type) {
    case "render":
      names = m.names
      render(m.html)
      schedule()
      break
    case "select":
      selected = m.id
      if (m.scroll) elementsOf(m.id)[0]?.scrollIntoView({ block: "nearest", inline: "nearest" })
      schedule()
      break
    case "hover":
      hovered = m.id
      schedule()
      break
    case "mode":
      mode = m.mode
      document.documentElement.dataset.mode = mode
      hovered = null
      schedule()
      break
    case "layout":
      root.className = m.center ? "flex min-h-svh w-full items-center justify-center p-10" : ""
      schedule()
      break
    case "theme":
      document.documentElement.classList.toggle("dark", m.dark)
      document.documentElement.style.colorScheme = m.dark ? "dark" : "light"
      break
  }
})

post({ type: "ready" })
