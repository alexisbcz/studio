// The studio: a visual editor for shadcn.go code. The Go source is the
// document; the engine (Go compiled to WebAssembly, in worker.js) renders
// it, inspects its calls and edits them; the canvas (preview.js) shows the
// result and reports what is clicked, hovered and dropped.

import * as CM from "./vendor/codemirror.js"
import { getDialog } from "./_shadcn/dialog.js"

const $ = (id) => document.getElementById(id)
const SNIPPET = "application/x-studio-snippet"
const STORE = "studio:doc"

const STARTER = `import (
	"github.com/alexisbcz/components/ui/button"
	"github.com/alexisbcz/components/ui/card"
	"github.com/alexisbcz/components/ui/input"
	"github.com/alexisbcz/components/ui/label"
	h "github.com/alexisbcz/libhtml"
)

// Page is what the studio renders: edit it here, or visually on the canvas.
func Page() h.Node {
	return h.Div(h.Class("flex min-h-svh items-center justify-center p-6"),
		card.Card(h.Class("w-full max-w-sm"),
			card.Header(
				card.Title("Login to your account"),
				card.Description("Enter your email below to login to your account"),
				card.Action(button.Button(button.VariantLink, "Sign Up")),
			),
			card.Content(
				h.Div(h.Class("flex flex-col gap-6"),
					h.Div(h.Class("grid gap-2"),
						label.Label(h.For("email"), "Email"),
						input.Input(h.ID("email"), h.Type("email"), h.Placeholder("m@example.com")),
					),
					h.Div(h.Class("grid gap-2"),
						label.Label(h.For("password"), "Password"),
						input.Input(h.ID("password"), h.Type("password")),
					),
				),
			),
			card.Footer(h.Class("flex-col gap-2"),
				button.Button(h.Class("w-full"), "Login"),
				button.Button(button.VariantOutline, h.Class("w-full"), "Login with Google"),
			),
		),
	)
}
`

const BLANK = `import (
	"github.com/alexisbcz/components/ui/button"
	h "github.com/alexisbcz/libhtml"
)

func Page() h.Node {
	return h.Div(h.Class("flex min-h-svh flex-col items-center justify-center gap-4 p-6"),
		h.H1(h.Class("text-2xl font-semibold tracking-tight"), "Hello, shadcn.go"),
		button.Button("Get started"),
	)
}
`

// --- state ---

const state = {
  code: "",
  style: "nova",
  base: "radix",
  nodes: [], // [{id, start, end, name}] of the last render
  tree: [],
  selected: null, // start offset of the selected call
  hover: null,
  mode: "select",
  center: false, // center the page on the canvas (examples)
  palette: [],
  examples: null,
}

// --- engine ---

class Engine {
  constructor() {
    this.seq = 0
    this.pending = new Map()
    this.start()
  }
  start() {
    this.worker?.terminate()
    this.worker = new Worker("worker.js")
    this.ready = new Promise((resolve, reject) => {
      this.worker.onmessage = (e) => {
        const m = e.data
        if (m.progress != null) return progress(m.progress)
        if (m.ready) return resolve()
        if (m.failed) return reject(new Error(m.failed))
        const p = this.pending.get(m.id)
        if (!p) return
        this.pending.delete(m.id)
        clearTimeout(p.timer)
        p.resolve({ ...m.result, ms: m.ms })
      }
    })
  }
  async call(fn, arg, timeout = 8000) {
    await this.ready
    const id = ++this.seq
    return new Promise((resolve) => {
      const timer = setTimeout(() => {
        // An endless loop in the program: start a new engine.
        this.pending.delete(id)
        for (const p of this.pending.values()) p.resolve({ error: "engine restarted" })
        this.pending.clear()
        this.start()
        resolve({ error: `${fn} took more than ${timeout / 1000}s and was stopped (an endless loop?)` })
      }, timeout)
      this.pending.set(id, { resolve, timer })
      this.worker.postMessage({ id, fn, arg })
    })
  }
}

const engine = new Engine()

function progress(p) {
  $("loading-bar").style.width = Math.round(p * 100) + "%"
  $("loading-text").textContent = p < 1 ? `Loading the Go engine… ${Math.round(p * 100)}%` : "Starting Go…"
}

// --- editor ---

const selectionMark = CM.Decoration.mark({ class: "cm-studio-selected" })
const setSelectionRange = CM.StateEffect.define()
const selectionField = CM.StateField.define({
  create: () => CM.Decoration.none,
  update(deco, tr) {
    deco = deco.map(tr.changes)
    for (const e of tr.effects) {
      if (e.is(setSelectionRange)) deco = e.value ? CM.Decoration.set([selectionMark.range(e.value.from, e.value.to)]) : CM.Decoration.none
    }
    return deco
  },
  provide: (f) => CM.EditorView.decorations.from(f),
})

const highlight = CM.HighlightStyle.define([
  { tag: CM.tags.keyword, color: "var(--cm-keyword)" },
  { tag: [CM.tags.string, CM.tags.special(CM.tags.string)], color: "var(--cm-string)" },
  { tag: CM.tags.comment, color: "var(--cm-comment)", fontStyle: "italic" },
  { tag: [CM.tags.number, CM.tags.bool, CM.tags.null], color: "var(--cm-number)" },
  { tag: [CM.tags.function(CM.tags.variableName), CM.tags.function(CM.tags.propertyName)], color: "var(--cm-function)" },
  { tag: [CM.tags.typeName, CM.tags.className], color: "var(--cm-type)" },
  { tag: CM.tags.propertyName, color: "var(--cm-property)" },
])

const editorTheme = CM.EditorView.theme({
  "&": { height: "100%", fontSize: "12.5px", backgroundColor: "var(--background)", color: "var(--foreground)" },
  ".cm-scroller": { fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace", lineHeight: "1.6" },
  ".cm-gutters": { backgroundColor: "var(--background)", color: "var(--muted-foreground)", border: "none" },
  ".cm-activeLine, .cm-activeLineGutter": { backgroundColor: "color-mix(in oklab, var(--muted) 60%, transparent)" },
  ".cm-studio-selected": { backgroundColor: "oklch(0.62 0.19 260 / 0.14)", borderRadius: "2px" },
  "&.cm-focused": { outline: "none" },
  ".cm-cursor": { borderLeftColor: "var(--foreground)" },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": { backgroundColor: "oklch(0.62 0.19 260 / 0.25) !important" },
  ".cm-tooltip": { backgroundColor: "var(--popover)", color: "var(--popover-foreground)", border: "1px solid var(--border)" },
})

let applyingExternal = false
const editor = new CM.EditorView({
  parent: $("editor"),
  state: CM.EditorState.create({
    doc: "",
    extensions: [
      CM.basicSetup,
      CM.go(),
      CM.indentUnit.of("\t"),
      CM.EditorState.tabSize.of(4),
      CM.keymap.of([CM.indentWithTab]),
      CM.syntaxHighlighting(highlight),
      editorTheme,
      selectionField,
      CM.EditorView.updateListener.of((u) => {
        if (u.docChanged) {
          state.code = u.state.doc.toString()
          if (state.selected != null) state.selected = u.changes.mapPos(state.selected, 1)
          if (!applyingExternal) scheduleRun()
          save()
        } else if (u.selectionSet && u.transactions.some((t) => t.isUserEvent("select"))) {
          selectAtCursor(u.state.selection.main.head)
        }
      }),
    ],
  }),
})

// setCode replaces the source, as one change of the differing middle, so
// that the history and the scroll position survive.
function setCode(code) {
  const old = editor.state.doc.toString()
  if (old === code) return
  let a = 0
  while (a < old.length && a < code.length && old[a] === code[a]) a++
  let b = 0
  while (b < old.length - a && b < code.length - a && old[old.length - 1 - b] === code[code.length - 1 - b]) b++
  applyingExternal = true
  editor.dispatch({ changes: { from: a, to: old.length - b, insert: code.slice(a, code.length - b) }, userEvent: "studio.edit" })
  applyingExternal = false
  state.code = code
}

// --- rendering ---

let runTimer = 0
let runSeq = 0
function scheduleRun(delay = 250) {
  clearTimeout(runTimer)
  runTimer = setTimeout(run, delay)
}

let previewReady
const previewLoaded = new Promise((r) => (previewReady = r))

async function run() {
  const seq = ++runSeq
  const res = await engine.call("run", { code: state.code, style: state.style, base: state.base, instrument: true })
  if (seq !== runSeq) return
  if (res.error) {
    showError(res.error)
    status(`Error · ${Math.round(res.ms ?? 0)} ms`)
    return
  }
  showError(null)
  state.nodes = res.nodes ?? []
  $("page-func").textContent = res.func + "()"
  const names = Object.fromEntries(state.nodes.map((n) => [n.id, n.name]))
  await previewLoaded
  frame().postMessage({ type: "render", html: res.html, names }, "*")
  status(`Rendered ${res.func}() in ${Math.round(res.ms)} ms · ${state.nodes.length} elements`)
  $("loading").hidden = true
}

function showError(msg) {
  const el = $("error")
  el.hidden = !msg
  el.textContent = msg ?? ""
}

function status(text) {
  $("status").textContent = text
}

const frame = () => $("preview").contentWindow

// --- selection ---

const nodeByID = (id) => state.nodes.find((n) => n.id === id)
const nodeAt = (start) => state.nodes.find((n) => n.start === start)
const selectedNode = () => (state.selected == null ? null : nodeAt(state.selected))

function select(start, { scroll = true, fromCode = false } = {}) {
  state.selected = start
  const node = selectedNode()
  frame().postMessage({ type: "select", id: node?.id ?? null, scroll }, "*")
  editor.dispatch({ effects: setSelectionRange.of(node ? { from: node.start, to: node.end } : null) })
  if (node && !fromCode && scroll && isCodeVisible()) {
    editor.dispatch({ effects: CM.EditorView.scrollIntoView(node.start, { y: "center" }) })
  }
  renderLayers()
  renderBreadcrumbs()
  inspect()
}

// selectAtCursor selects the innermost call around the editor's cursor.
function selectAtCursor(pos) {
  let best = null
  for (const n of state.nodes) {
    if (n.start <= pos && pos <= n.end && (!best || n.end - n.start < best.end - best.start)) best = n
  }
  if (best && best.start !== state.selected) select(best.start, { fromCode: true })
}

function isCodeVisible() {
  const panel = document.querySelector('[data-panel="code"]')
  return panel && !panel.hidden
}

// --- layers ---

function parentsOf(id) {
  const path = []
  const walk = (items, trail) => {
    for (const it of items) {
      if (it.id === id) return path.push(...trail, it), true
      if (walk(it.children, [...trail, it])) return true
    }
    return false
  }
  walk(state.tree, [])
  return path
}

function displayName(name) {
  if (!name) return "?"
  const [pkg, fn] = name.split(".")
  if (pkg === "h") return fn.toLowerCase()
  return name
}

function renderLayers() {
  const root = $("layers")
  const sel = selectedNode()?.id
  const frag = document.createDocumentFragment()
  const seen = new Set()
  const walk = (items, depth) => {
    for (const it of items) {
      const node = nodeByID(it.id)
      // Loops render a call several times: list it once.
      const key = it.id + ":" + depth
      if (seen.has(key)) continue
      seen.add(key)
      const row = document.createElement("button")
      row.type = "button"
      row.dataset.layer = it.id
      row.role = "treeitem"
      row.className =
        "flex w-full min-w-0 items-center gap-1.5 rounded-md py-1 pr-2 text-left text-xs hover:bg-accent " +
        (it.id === sel ? "bg-accent text-accent-foreground font-medium " : "") +
        (it.hidden ? "opacity-60 " : "")
      row.style.paddingLeft = 6 + depth * 12 + "px"
      const name = document.createElement("span")
      name.className = "shrink-0 font-mono"
      name.textContent = displayName(node?.name)
      row.append(name)
      if (it.text) {
        const t = document.createElement("span")
        t.className = "truncate text-muted-foreground"
        t.textContent = it.text
        row.append(t)
      }
      row.addEventListener("click", () => node && select(node.start))
      row.addEventListener("mouseenter", () => frame().postMessage({ type: "hover", id: it.id }, "*"))
      row.addEventListener("mouseleave", () => frame().postMessage({ type: "hover", id: null }, "*"))
      row.addEventListener("dragover", (e) => {
        if (!e.dataTransfer.types.includes(SNIPPET)) return
        e.preventDefault()
        row.classList.add("ring-2", "ring-ring")
      })
      row.addEventListener("dragleave", () => row.classList.remove("ring-2", "ring-ring"))
      row.addEventListener("drop", (e) => {
        e.preventDefault()
        row.classList.remove("ring-2", "ring-ring")
        if (node) insertSnippet(Number(e.dataTransfer.getData(SNIPPET)), node.start, "inside")
      })
      frag.append(row)
      walk(it.children, depth + 1)
    }
  }
  walk(state.tree, 0)
  if (!state.tree.length) {
    const empty = document.createElement("p")
    empty.className = "p-2 text-xs text-muted-foreground"
    empty.textContent = "Nothing rendered yet."
    frag.append(empty)
  }
  root.replaceChildren(frag)
  root.querySelector(`[data-layer="${sel}"]`)?.scrollIntoView({ block: "nearest" })
}

function renderBreadcrumbs() {
  const el = $("breadcrumbs")
  const node = selectedNode()
  el.replaceChildren()
  if (!node) return
  const path = parentsOf(node.id)
  path.forEach((it, i) => {
    if (i) el.append(Object.assign(document.createElement("span"), { textContent: "›" }))
    const b = document.createElement("button")
    b.type = "button"
    b.className = "shrink-0 truncate rounded px-1 font-mono hover:bg-accent hover:text-accent-foreground" + (i === path.length - 1 ? " text-foreground" : "")
    b.textContent = displayName(nodeByID(it.id)?.name)
    b.addEventListener("click", () => select(nodeByID(it.id).start))
    el.append(b)
  })
}

// --- inspector ---

const proto = (name) => $("proto-" + name).content.firstElementChild.cloneNode(true)

function section(title, ...children) {
  const s = document.createElement("section")
  s.className = "flex flex-col gap-2 border-b p-3"
  if (title) {
    const h = document.createElement("h3")
    h.className = "text-xs font-medium"
    h.textContent = title
    s.append(h)
  }
  s.append(...children)
  return s
}

function field(labelText, control) {
  const wrap = document.createElement("div")
  wrap.className = "grid gap-1"
  if (labelText) {
    const l = proto("label")
    l.textContent = labelText
    wrap.append(l)
  }
  wrap.append(control)
  return wrap
}

function selectControl(options, value, onChange, empty) {
  const wrap = proto("select")
  const sel = wrap.querySelector("select") ?? wrap
  sel.replaceChildren()
  if (empty) sel.append(new Option(empty, ""))
  for (const o of options) sel.append(new Option(o.split(".").pop(), o, false, o === value))
  if (empty && !value) sel.value = ""
  sel.addEventListener("change", () => onChange(sel.value))
  return wrap
}

// optionLabel names a variant from its constants: button.VariantOutline -> Variant.
function optionLabel(options) {
  const names = options.map((o) => o.split(".").pop())
  let p = names[0]
  for (const n of names) while (!n.startsWith(p)) p = p.slice(0, -1)
  const m = p.match(/^[A-Z][a-z]*/)
  return m ? m[0] : "Option"
}

let inspectSeq = 0
async function inspect() {
  const box = $("inspector")
  const node = selectedNode()
  if (!node) {
    box.replaceChildren(emptyInspector)
    return
  }
  const seq = ++inspectSeq
  const info = await engine.call("inspect", { code: state.code, start: node.start })
  if (seq !== inspectSeq) return
  if (info.error) {
    box.replaceChildren(section(null, Object.assign(document.createElement("p"), { className: "text-xs text-destructive", textContent: info.error })))
    return
  }
  const parts = []

  // Header: the call and its actions.
  const head = document.createElement("div")
  head.className = "flex items-center justify-between gap-2"
  const title = document.createElement("div")
  title.className = "min-w-0"
  title.innerHTML = `<div class="truncate font-mono text-sm font-medium"></div><div class="text-xs text-muted-foreground"></div>`
  title.children[0].textContent = info.name
  title.children[1].textContent = `Line ${info.line}`
  const actions = proto("actions")
  actions.addEventListener("click", (e) => {
    const a = e.target.closest("[data-action]")
    if (a) action(a.dataset.action)
  })
  head.append(title, actions)
  parts.push(section(null, head))

  // Content: the strings.
  const texts = info.args.filter((a) => a.kind === "text")
  if (texts.length) {
    parts.push(section("Content", ...texts.map((a) => {
      const long = a.value.length > 40 || a.value.includes("\n")
      const control = long ? proto("textarea") : proto("input")
      const input = control.matches("input, textarea") ? control : control.querySelector("input, textarea")
      input.value = a.value
      commitOnChange(input, (v) => edit({ op: "replace", start: a.start, end: a.end, text: JSON.stringify(v) }))
      return control
    })))
  }

  // Variants: the typed constants given, and those the call could take.
  const options = info.args.filter((a) => a.kind === "option")
  const choices = info.choices ?? []
  if (options.length || choices.length) {
    const fields = options.map((a) =>
      field(optionLabel(a.options), selectControl(a.options, a.value, (v) => edit({ op: "replace", start: a.start, end: a.end, text: v }))),
    )
    for (const c of choices) {
      fields.push(field(c.type, selectControl(c.options, "", (v) => v && edit({ op: "addArg", call: node.start, text: v }), "Default")))
    }
    parts.push(section("Variants", ...fields))
  }

  // Classes.
  const cls = info.args.find((a) => a.kind === "class")
  const classArea = proto("textarea")
  classArea.value = cls?.value ?? ""
  classArea.placeholder = "Tailwind classes"
  commitOnChange(classArea, (v) => edit({ op: "setClass", call: node.start, text: v.trim().replace(/\s+/g, " ") }))
  parts.push(section("Classes", classArea))

  // Children.
  const children = info.args.filter((a) => a.kind === "child")
  if (children.length) {
    const list = document.createElement("div")
    list.className = "flex flex-col gap-1"
    for (const c of children) {
      const chip = proto("chip")
      chip.textContent = c.name
      chip.addEventListener("click", () => select(c.start))
      list.append(chip)
    }
    parts.push(section("Children", list))
  }

  // Other arguments, as code.
  const others = info.args.filter((a) => a.kind === "code")
  if (others.length) {
    parts.push(section("Arguments", ...others.map((a) => {
      const input = proto("input")
      input.classList.add("font-mono")
      input.value = a.value
      commitOnChange(input, (v) => edit({ op: "replace", start: a.start, end: a.end, text: v }))
      return input
    })))
  }

  const add = proto("chip")
  add.textContent = "+ Add a component inside"
  add.classList.remove("font-mono")
  add.addEventListener("click", () => showTab("left", "add"))
  parts.push(section(null, add))

  box.replaceChildren(...parts)
}

const emptyInspector = $("inspector").firstElementChild

// commitOnChange calls fn with the control's value on Enter (not in a
// textarea), on blur and after a pause in typing.
function commitOnChange(input, fn) {
  const initial = input.value
  let last = initial
  let timer = 0
  const commit = () => {
    clearTimeout(timer)
    if (input.value !== last) {
      last = input.value
      fn(input.value)
    }
  }
  input.addEventListener("input", () => {
    clearTimeout(timer)
    timer = setTimeout(commit, 600)
  })
  input.addEventListener("blur", commit)
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && (input.tagName !== "TEXTAREA" || e.metaKey || e.ctrlKey)) {
      e.preventDefault()
      commit()
    }
    if (e.key === "Escape") {
      input.value = last
      input.blur()
    }
  })
}

// --- edits ---

async function edit(e) {
  const res = await engine.call("apply", { code: state.code, edit: e })
  if (res.error) {
    status(res.error)
    return false
  }
  setCode(res.src) // maps the selection through the change
  if (res.select >= 0) state.selected = res.select
  else if (e.op === "delete") state.selected = null
  clearTimeout(runTimer)
  await run()
  return true
}

async function action(name) {
  const node = selectedNode()
  if (!node) return
  switch (name) {
    case "up":
    case "down":
      return edit({ op: "move", call: node.start, delta: name === "up" ? -1 : 1 })
    case "duplicate":
      return edit({ op: "insert", call: node.start, where: "after", text: state.code.slice(node.start, node.end) })
    case "delete":
      return edit({ op: "delete", call: node.start })
    case "code":
      showTab("right", "code")
      editor.dispatch({ selection: { anchor: node.start, head: node.end }, effects: CM.EditorView.scrollIntoView(node.start, { y: "center" }) })
      editor.focus()
  }
}

async function insertSnippet(index, call, where) {
  const s = state.palette[index]
  if (!s) return
  if (call == null) {
    call = rootNode()?.start
    where = "inside"
    if (call == null) return status("Nothing to add into: render something first.")
  }
  await edit({ op: "insert", call, where, text: s.code, imports: s.imports })
  status(`Added ${s.label}`)
}

// rootNode is the outermost element of the page.
function rootNode() {
  const first = state.tree[0]
  return first ? nodeByID(first.id) : null
}

// --- tabs ---

// The side panels: the tabs switch panels that stay mounted.
for (const root of document.querySelectorAll('[data-slot="tabs"]')) {
  root.addEventListener("shadcn:value-change", (e) => setPanel(root, e.detail.value))
}

function setPanel(tabsRoot, value) {
  for (const p of tabsRoot.parentElement.querySelectorAll(":scope > [data-panel]")) p.hidden = p.dataset.panel !== value
  if (value === "code") editor.requestMeasure()
}

function showTab(side, value) {
  const trigger = $(side)?.querySelector(`[role="tab"][id$="-trigger-${value}"]`)
  if (trigger && trigger.getAttribute("data-state") !== "active") trigger.click()
}

// --- documents ---

async function share() {
  const json = JSON.stringify({ code: state.code, style: state.style, base: state.base, center: state.center })
  const stream = new Blob([json]).stream().pipeThrough(new CompressionStream("deflate-raw"))
  const bytes = new Uint8Array(await new Response(stream).arrayBuffer())
  let bin = ""
  for (const b of bytes) bin += String.fromCharCode(b)
  const hash = "#c=" + btoa(bin).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "")
  history.replaceState(null, "", hash)
  await navigator.clipboard?.writeText(location.href).catch(() => {})
  status("Share link copied to the clipboard")
}

async function fromHash() {
  const m = location.hash.match(/^#c=([\w-]+)$/)
  if (!m) return null
  try {
    const bin = atob(m[1].replaceAll("-", "+").replaceAll("_", "/"))
    const bytes = Uint8Array.from(bin, (c) => c.charCodeAt(0))
    const stream = new Blob([bytes]).stream().pipeThrough(new DecompressionStream("deflate-raw"))
    return JSON.parse(await new Response(stream).text())
  } catch {
    return null
  }
}

function save() {
  try {
    localStorage.setItem(STORE, JSON.stringify({ code: state.code, style: state.style, base: state.base, center: state.center }))
  } catch {}
}

function load() {
  try {
    return JSON.parse(localStorage.getItem(STORE))
  } catch {
    return null
  }
}

function download() {
  const header = `// Made with shadcn.go studio (https://alexisbcz.github.io/studio/).
// Style: ${state.style}, primitives: ${state.base}. Render it with
// style.Scope(style.${state.style[0].toUpperCase() + state.style.slice(1)}, Page()) (github.com/alexisbcz/components/style).

`
  const code = /^\s*package\s/.test(state.code) ? state.code : "package main\n\n" + state.code
  const a = document.createElement("a")
  a.href = URL.createObjectURL(new Blob([header + code], { type: "text/x-go" }))
  a.download = "page.go"
  a.click()
  URL.revokeObjectURL(a.href)
}

function setCenter(on) {
  state.center = on
  const t = $("center")
  t.setAttribute("aria-pressed", String(on))
  t.setAttribute("data-state", on ? "on" : "off")
  frame()?.postMessage({ type: "layout", center: on }, "*")
}

function setDocument(doc) {
  setCenter(!!doc.center)
  if (doc.style) state.style = doc.style
  if (doc.base) state.base = doc.base
  $("style").value = state.style
  $("base").value = state.base
  state.selected = null
  setCode(doc.code)
  select(null, { scroll: false })
  scheduleRun(0)
}

async function openExample(name) {
  if (!state.examples) {
    status("Loading the examples…")
    state.examples = await (await fetch("examples.json")).json()
  }
  const code = state.examples[name]
  if (!code) return
  const base = name.startsWith("aria--") ? "aria" : name.startsWith("base--") ? "base" : "radix"
  setDocument({ code, base, style: state.style, center: true })
  status(`Opened ${name}`)
}

// --- wiring ---

addEventListener("message", (e) => {
  if (e.source !== frame()) return
  const m = e.data
  switch (m.type) {
    case "ready":
      previewReady()
      frame().postMessage({ type: "theme", dark: document.documentElement.classList.contains("dark") }, "*")
      frame().postMessage({ type: "mode", mode: state.mode }, "*")
      frame().postMessage({ type: "layout", center: state.center }, "*")
      break
    case "tree":
      state.tree = m.tree
      if (state.selected != null && !selectedNode()) state.selected = null
      select(state.selected, { scroll: false })
      break
    case "select":
      select(m.id == null ? null : nodeByID(m.id)?.start ?? null)
      break
    case "hover":
      for (const r of $("layers").querySelectorAll("[data-layer]")) r.classList.toggle("bg-accent/60", Number(r.dataset.layer) === m.id)
      break
    case "drop":
      insertSnippet(m.snippet, nodeByID(m.id)?.start, m.where)
      break
    case "key":
      onKey(m)
      break
  }
})

function onKey(k) {
  if (k.meta && k.key.toLowerCase() === "k") return openExamples()
  if (k.meta && k.key.toLowerCase() === "z") return k.shift ? CM.redo(editor) : CM.undo(editor)
  if (k.meta && k.key.toLowerCase() === "s") return formatCode()
  if (k.meta && k.key.toLowerCase() === "d") return action("duplicate")
  switch (k.key) {
    case "Delete":
    case "Backspace":
      return action("delete")
    case "Escape":
      return select(null)
    case "ArrowUp":
    case "ArrowDown":
      if (k.alt) return action(k.key === "ArrowUp" ? "up" : "down")
      return selectSibling(k.key === "ArrowUp" ? -1 : 1)
  }
}

// selectSibling moves the selection through the layers in order.
function selectSibling(delta) {
  const flat = []
  const walk = (items) => items.forEach((it) => (flat.push(it.id), walk(it.children)))
  walk(state.tree)
  const ids = [...new Set(flat)]
  const i = ids.indexOf(selectedNode()?.id)
  const next = ids[Math.max(0, Math.min(ids.length - 1, i + delta))]
  if (next != null) select(nodeByID(next).start)
}

async function formatCode() {
  const res = await engine.call("format", { code: state.code })
  if (!res.error) setCode(res.src)
  status(res.error ?? "Formatted")
}

function openExamples() {
  // Focus leaves the canvas first: the dialog focuses its search field.
  if (document.activeElement === $("preview")) $("preview").blur()
  window.focus()
  getDialog($("examples-trigger"))?.setOpen(true)
  focusSearch()
}

// focusSearch focuses the examples' search field once the dialog shows it.
function focusSearch() {
  let tries = 0
  const tick = () => {
    const input = document.querySelector('[role="dialog"] [data-slot="command-input"]')
    if (input) return input.focus()
    if (++tries < 20) requestAnimationFrame(tick)
  }
  requestAnimationFrame(tick)
}
$("examples-trigger").addEventListener("click", focusSearch)

document.addEventListener("keydown", (e) => {
  const inField = e.target instanceof Element && e.target.closest("input, textarea, select, .cm-editor, [contenteditable]")
  const meta = e.metaKey || e.ctrlKey
  if (meta && e.key.toLowerCase() === "k") {
    e.preventDefault()
    return openExamples()
  }
  if (meta && e.key.toLowerCase() === "s") {
    e.preventDefault()
    return formatCode()
  }
  if (inField) return
  if ((meta && ["z", "d"].includes(e.key.toLowerCase())) || ["Delete", "Backspace", "Escape"].includes(e.key) || (["ArrowUp", "ArrowDown"].includes(e.key) && state.selected != null)) {
    e.preventDefault()
    onKey({ key: e.key, meta, shift: e.shiftKey, alt: e.altKey })
  }
})

// The examples palette: the command items dispatch cmdk-item-select.
document.addEventListener(
  "cmdk-item-select",
  (e) => {
    const name = e.target.closest?.("[data-example]")?.dataset.example
    if (!name) return
    getDialog($("examples-trigger"))?.setOpen(false)
    openExample(name)
  },
  true,
)

$("style").addEventListener("change", (e) => {
  state.style = e.target.value
  save()
  scheduleRun(0)
})
$("base").addEventListener("change", (e) => {
  state.base = e.target.value
  save()
  scheduleRun(0)
})

const widths = { desktop: "100%", tablet: "768px", mobile: "390px" }
$("viewport").addEventListener("click", (e) => {
  const v = e.target.closest("[data-pick]")?.dataset.pick
  if (v && widths[v]) $("frame").style.width = widths[v]
})
$("mode").addEventListener("click", (e) => {
  const v = e.target.closest("[data-pick]")?.dataset.pick
  if (v === "select" || v === "interact") {
    state.mode = v
    frame().postMessage({ type: "mode", mode: v }, "*")
  }
})

$("center").addEventListener("click", () => {
  // The toggle's own behavior flips aria-pressed; follow it.
  setTimeout(() => {
    setCenter($("center").getAttribute("aria-pressed") === "true")
    save()
  })
})

$("theme").addEventListener("click", () => {
  const dark = !document.documentElement.classList.contains("dark")
  document.documentElement.classList.toggle("dark", dark)
  try {
    localStorage.setItem("studio:theme", dark ? "dark" : "light")
  } catch {}
  frame().postMessage({ type: "theme", dark }, "*")
})
$("share").addEventListener("click", share)
$("download").addEventListener("click", download)
$("new-page").addEventListener("click", () => {
  history.replaceState(null, "", location.pathname)
  setDocument({ code: BLANK })
})

// Palette: click adds inside the selection (or the page); drag onto the
// canvas or the layers.
$("palette").addEventListener("click", (e) => {
  const b = e.target.closest("[data-snippet]")
  if (b) insertSnippet(Number(b.dataset.snippet), state.selected, "inside")
})
$("palette").addEventListener("dragstart", (e) => {
  const b = e.target.closest("[data-snippet]")
  if (!b) return
  e.dataTransfer.setData(SNIPPET, b.dataset.snippet)
  e.dataTransfer.setData("text/plain", state.palette[Number(b.dataset.snippet)]?.code ?? "")
  e.dataTransfer.effectAllowed = "copy"
})

// --- start ---

;(async () => {
  fetch("palette.json").then((r) => r.json()).then((p) => (state.palette = p))
  const doc = (await fromHash()) ?? load() ?? { code: STARTER }
  setDocument({ code: doc.code || STARTER, style: doc.style || "nova", base: doc.base || "radix", center: doc.center })
  try {
    await engine.ready
  } catch (err) {
    $("loading-text").textContent = "The Go engine failed to load: " + err.message
    return
  }
  status("Ready")
})()
