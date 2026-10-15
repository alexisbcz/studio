codemirror.js bundles CodeMirror 6 (codemirror 6.0.2, @codemirror/lang-go
6.0.1, @codemirror/state 6.7.6, @codemirror/view 6.43.14,
@codemirror/language 6.13.1, @lezer/go 1.0.1, @lezer/highlight 1.2.5), MIT
licensed, built with esbuild 0.25.11 from:

```js
export { EditorView, keymap, Decoration, ViewPlugin } from "@codemirror/view"
export { EditorState, StateField, StateEffect, EditorSelection } from "@codemirror/state"
export { basicSetup } from "codemirror"
export { go } from "@codemirror/lang-go"
export { indentUnit, HighlightStyle, syntaxHighlighting } from "@codemirror/language"
export { indentWithTab, undo, redo } from "@codemirror/commands"
export { tags } from "@lezer/highlight"
```

`esbuild entry.js --bundle --format=esm --minify --legal-comments=eof`
