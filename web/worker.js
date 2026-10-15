// Runs the engine (studio.wasm, the Go program in cmd/wasm) off the main
// thread. Messages: {id, fn, arg} -> {id, result}; while loading,
// {progress: 0..1}, then {ready: true}.

importScripts("wasm_exec.js")

async function load() {
  const manifest = await (await fetch("studio.json", { cache: "no-cache" })).json()
  const res = await fetch("studio.wasm?v=" + manifest.version)
  if (!res.ok) throw new Error("studio.wasm: " + res.status)
  // Progress against the uncompressed size: servers compress the module.
  let loaded = 0
  const reader = res.body.getReader()
  const stream = new ReadableStream({
    async pull(ctrl) {
      const { done, value } = await reader.read()
      if (done) return ctrl.close()
      loaded += value.byteLength
      postMessage({ progress: Math.min(loaded / manifest.wasmSize, 1) })
      ctrl.enqueue(value)
    },
  })
  const go = new Go()
  const { instance } = await WebAssembly.instantiateStreaming(
    new Response(stream, { headers: { "Content-Type": "application/wasm" } }),
    go.importObject,
  )
  go.run(instance)
  postMessage({ ready: true })
}

const ready = load().catch((err) => postMessage({ failed: String(err) }))

onmessage = async (e) => {
  await ready
  const { id, fn, arg } = e.data
  const t = performance.now()
  let result
  try {
    result = JSON.parse(globalThis.studioEngine[fn](JSON.stringify(arg)))
  } catch (err) {
    result = { error: String(err) }
  }
  postMessage({ id, result, ms: performance.now() - t })
}
