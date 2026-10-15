package main

import (
	"sort"
	"strings"

	"github.com/alexisbcz/components/lib"
	"github.com/alexisbcz/components/style"
	"github.com/alexisbcz/components/ui/badge"
	"github.com/alexisbcz/components/ui/button"
	"github.com/alexisbcz/components/ui/command"
	"github.com/alexisbcz/components/ui/dialog"
	"github.com/alexisbcz/components/ui/input"
	"github.com/alexisbcz/components/ui/kbd"
	"github.com/alexisbcz/components/ui/label"
	nativeselect "github.com/alexisbcz/components/ui/native-select"
	"github.com/alexisbcz/components/ui/separator"
	"github.com/alexisbcz/components/ui/tabs"
	"github.com/alexisbcz/components/ui/toggle"
	"github.com/alexisbcz/components/ui/textarea"
	tg "github.com/alexisbcz/components/ui/toggle-group"
	"github.com/alexisbcz/components/ui/tooltip"
	"github.com/alexisbcz/icons/lucide"
	h "github.com/alexisbcz/libhtml"
	"github.com/alexisbcz/studio/engine"
)

// page is the studio: built with shadcn.go itself.
func page(examples []string, version string) h.Node {
	return h.Doctype(
		h.Html(h.Lang("en"),
			h.Head(
				h.Meta(h.Charset("utf-8")),
				h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
				h.Title("shadcn.go studio"),
				h.Meta(h.Name("description"), h.Content("A visual editor for shadcn.go: design Go user interfaces in the browser, with Go running in WebAssembly.")),
				h.Link(h.Rel("icon"), h.Href("favicon.svg"), h.Type("image/svg+xml")),
				h.Script(h.UnsafeJS(themeScript)),
				h.Link(h.Rel("stylesheet"), h.Href("studio.css?v="+version)),
				h.Link(h.Rel("modulepreload"), h.Href("vendor/codemirror.js")),
				h.Script(h.Type("module"), h.Src("_shadcn/index.js?v="+version)),
				h.Script(h.Type("module"), h.Src("studio.js?v="+version)),
			),
			h.Body(h.Class("h-svh overflow-hidden overscroll-none bg-background text-foreground antialiased"),
				style.Scope(style.Nova,
					h.Div(h.Class("flex h-svh flex-col"),
						header(examples),
						h.Div(h.Class("grid min-h-0 flex-1 grid-cols-1 md:grid-cols-[15rem_minmax(0,1fr)] lg:grid-cols-[15rem_minmax(0,1fr)_24rem]"),
							leftPanel(),
							canvas(),
							rightPanel(),
						),
						statusBar(),
					),
					protos(),
				),
			),
		),
	)
}

// themeScript applies the saved theme before the first paint.
const themeScript = `try{if(localStorage.getItem("studio:theme")==="dark"||(!localStorage.getItem("studio:theme")&&matchMedia("(prefers-color-scheme: dark)").matches))document.documentElement.classList.add("dark")}catch(_){}`

func iconButton(label string, icon func(...any) *h.Element, attrs ...any) h.Node {
	return tooltip.Tooltip(
		tooltip.Trigger(lib.AsChild, button.Button(append([]any{button.VariantGhost, button.SizeIconSm, h.Aria("label", label), icon()}, attrs...)...)),
		tooltip.Content(label),
	)
}

func header(examples []string) h.Node {
	return h.Header(h.Class("flex h-12 shrink-0 items-center gap-2 border-b px-3"),
		h.A(h.Href("./"), h.Class("flex items-center gap-2 font-medium"),
			h.Span(h.Class("flex size-6 items-center justify-center rounded-md bg-primary text-primary-foreground"), lucide.Blocks(h.Class("size-3.5"))),
			h.Span(h.Class("hidden sm:inline"), "shadcn.go ", h.Span(h.Class("text-muted-foreground"), "studio")),
		),
		separator.Separator(separator.Vertical, h.Class("mx-1 data-[orientation=vertical]:h-5")),
		examplesDialog(examples),
		button.Button(button.VariantGhost, button.SizeSm, h.ID("new-page"), lucide.Plus(), "New"),
		h.Div(h.Class("flex-1")),
		h.Div(h.Class("hidden items-center gap-2 md:flex"),
			nativeselect.NativeSelect(h.ID("style"), h.Aria("label", "Style"), h.Class("h-8 w-28 text-xs"),
				h.Map(style.All, func(s style.Style) h.Node {
					return nativeselect.Option(h.Value(string(s)), h.If(s == style.Nova, h.Bool("selected")), strings.ToUpper(string(s)[:1])+string(s)[1:])
				}),
			),
			nativeselect.NativeSelect(h.ID("base"), h.Aria("label", "Primitives"), h.Class("h-8 w-32 text-xs"),
				nativeselect.Option(h.Value("radix"), "Radix UI"),
				nativeselect.Option(h.Value("base"), "Base UI"),
				nativeselect.Option(h.Value("aria"), "React Aria"),
			),
			tg.ToggleGroup(tg.TypeSingle, tg.VariantOutline, tg.SizeSm, tg.Value("desktop"), h.ID("viewport"), h.Aria("label", "Viewport"),
				tg.Item(tg.Value("desktop"), h.Data("pick", "desktop"), h.Aria("label", "Desktop"), lucide.Monitor()),
				tg.Item(tg.Value("tablet"), h.Data("pick", "tablet"), h.Aria("label", "Tablet"), lucide.Tablet()),
				tg.Item(tg.Value("mobile"), h.Data("pick", "mobile"), h.Aria("label", "Mobile"), lucide.Smartphone()),
			),
		),
		separator.Separator(separator.Vertical, h.Class("mx-1 hidden data-[orientation=vertical]:h-5 md:block")),
		iconButton("Toggle theme", lucide.Moon, h.ID("theme")),
		iconButton("Copy share link", lucide.Share2, h.ID("share")),
		iconButton("Download page.go", lucide.Download, h.ID("download")),
		button.Button(button.VariantGhost, button.SizeIconSm, lib.AsChild,
			h.A(h.Href("https://github.com/alexisbcz/studio"), h.Aria("label", "GitHub"), h.Target("_blank"), h.Rel("noreferrer"), lucide.Github()),
		),
	)
}

// examplesDialog is the ⌘K picker of the 861 docs examples.
func examplesDialog(names []string) h.Node {
	groups := map[string][]string{}
	for _, n := range names {
		groups[component(n)] = append(groups[component(n)], n)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return dialog.Dialog(
		dialog.Trigger(lib.AsChild,
			button.Button(button.VariantOutline, button.SizeSm, h.ID("examples-trigger"), h.Class("text-muted-foreground"),
				lucide.Search(), h.Span(h.Class("hidden sm:inline"), "Examples"),
				kbd.Group(h.Class("ml-2 hidden sm:inline-flex"), kbd.Kbd("⌘"), kbd.Kbd("K")),
			),
		),
		dialog.Content(h.Class("overflow-hidden p-0 sm:max-w-lg"), dialog.ShowCloseButton(false), h.ID("examples-dialog"),
			dialog.Header(h.Class("sr-only"),
				dialog.Title("Examples"),
				dialog.Description("Open one of the shadcn/ui docs examples, ported to Go."),
			),
			command.Command(
				command.Input(h.Placeholder("Search 861 examples...")),
				command.List(h.Class("max-h-[min(28rem,60vh)]"),
					command.Empty("No examples found."),
					h.Map(keys, func(k string) h.Node {
						return command.Group(command.Heading(k),
							h.Map(groups[k], func(n string) h.Node {
								return command.Item(command.Value(n), h.Data("example", n), n, exampleIcon(n))
							}),
						)
					}),
				),
			),
		),
	)
}

func exampleIcon(name string) h.Node {
	switch {
	case strings.HasPrefix(name, "aria--"):
		return h.Span(h.Class("rounded bg-muted px-1 text-[10px] text-muted-foreground"), "Aria")
	case strings.HasPrefix(name, "base--"):
		return h.Span(h.Class("rounded bg-muted px-1 text-[10px] text-muted-foreground"), "Base")
	}
	return nil
}

// component is the component an example belongs to ("button" for
// "nova--button-demo").
func component(name string) string {
	for _, p := range []string{"aria--", "base--", "nova--"} {
		name = strings.TrimPrefix(name, p)
	}
	for _, c := range components {
		if name == c || strings.HasPrefix(name, c+"-") {
			return c
		}
	}
	if i := strings.Index(name, "-"); i > 0 {
		return name[:i]
	}
	return name
}

// components are the component names, longest first, so that
// "button-group-demo" belongs to button-group, not button.
var components = func() []string {
	c := strings.Fields(`accordion alert-dialog alert aspect-ratio attachment avatar badge breadcrumb bubble
		button-group button calendar card carousel chart checkbox collapsible combobox command context-menu
		data-table date-picker dialog direction drawer dropdown-menu empty field form hover-card input-group
		input-otp input item kbd label marker menubar message-scroller message native-select navigation-menu
		pagination popover progress questionnaire radio-group resizable scroll-area select separator sheet
		sidebar skeleton slider sonner spinner switch table tabs textarea toast toggle-group toggle tooltip typography`)
	sort.Slice(c, func(i, j int) bool { return len(c[i]) > len(c[j]) })
	return c
}()

// panelTabs are tabs whose panels stay mounted (tabs.Content remounts
// its children, which would lose the editor's state): the tabs only switch
// the panels (studio.js, on shadcn:value-change).
func panelTabs(value string, triggers []h.Node, panels ...panel) h.Node {
	return h.Div(h.Class("flex min-h-0 flex-1 flex-col"),
		tabs.Tabs(tabs.Value(value), h.Class("border-b p-2"),
			tabs.List(h.Class("w-full"), h.Group(nodes(triggers)...)),
		),
		h.Map(panels, func(p panel) h.Node {
			return h.Div(h.Data("panel", p.value), h.Class(p.class), h.If(p.value != value, h.Hidden()), p.content)
		}),
	)
}

type panel struct {
	value, class string
	content      h.Node
}

func nodes(ns []h.Node) []any {
	out := make([]any, len(ns))
	for i, n := range ns {
		out[i] = n
	}
	return out
}

func leftPanel() h.Node {
	return h.Aside(h.Class("hidden min-h-0 flex-col border-r md:flex"), h.ID("left"),
		panelTabs("layers", []h.Node{
			tabs.Trigger(tabs.Value("layers"), lucide.Layers(), "Layers"),
			tabs.Trigger(tabs.Value("add"), lucide.Plus(), "Add"),
		},
			panel{"layers", "min-h-0 flex-1 overflow-auto p-2",
				h.Div(h.ID("layers"), h.Class("text-sm"), h.Role("tree"), h.Aria("label", "Layers")),
			},
			panel{"add", "min-h-0 flex-1 overflow-auto p-2", h.Group(
				h.P(h.Class("px-1 pb-2 text-xs text-muted-foreground"), "Click to add inside the selected element, or drag onto the canvas."),
				palette(),
			)},
		),
	)
}

func palette() h.Node {
	var groups []string
	byGroup := map[string][]int{}
	for i, s := range engine.Palette {
		if _, ok := byGroup[s.Group]; !ok {
			groups = append(groups, s.Group)
		}
		byGroup[s.Group] = append(byGroup[s.Group], i)
	}
	return h.Div(h.ID("palette"), h.Class("flex flex-col gap-3"),
		h.Map(groups, func(g string) h.Node {
			return h.Div(
				h.Div(h.Class("px-1 pb-1 text-xs font-medium text-muted-foreground"), g),
				h.Div(h.Class("grid grid-cols-2 gap-1"),
					h.Map(byGroup[g], func(i int) h.Node {
						return h.Button(h.Type("button"), h.Data("snippet", itoa(i)), h.Attr("draggable", "true"),
							h.Class("cursor-grab truncate rounded-md border bg-card px-2 py-1.5 text-left text-xs hover:bg-accent hover:text-accent-foreground active:cursor-grabbing"),
							engine.Palette[i].Label)
					}),
				),
			)
		}),
	)
}

func canvas() h.Node {
	return h.Main(h.Class("relative flex min-h-0 min-w-0 flex-col bg-muted/50"),
		h.Div(h.Class("flex h-10 shrink-0 items-center gap-2 border-b bg-background px-2"),
			tg.ToggleGroup(tg.TypeSingle, tg.SizeSm, tg.Value("select"), h.ID("mode"), h.Aria("label", "Mode"),
				tg.Item(tg.Value("select"), h.Data("pick", "select"), h.Aria("label", "Select elements"), lucide.MousePointer2(), h.Span(h.Class("text-xs"), "Select")),
				tg.Item(tg.Value("interact"), h.Data("pick", "interact"), h.Aria("label", "Use the page"), lucide.Hand(), h.Span(h.Class("text-xs"), "Interact")),
			),
			toggle.Toggle(toggle.SizeSm, h.ID("center"), h.Aria("label", "Center the page"), h.Title("Center the page on the canvas"),
				lucide.Crosshair(), h.Span(h.Class("text-xs"), "Center"),
			),
			separator.Separator(separator.Vertical, h.Class("data-[orientation=vertical]:h-5")),
			h.Div(h.ID("breadcrumbs"), h.Class("flex min-w-0 flex-1 items-center gap-1 overflow-hidden text-xs text-muted-foreground")),
			h.Span(h.ID("page-func"), h.Class("font-mono text-xs text-muted-foreground")),
		),
		h.Div(h.ID("stage"), h.Class("relative min-h-0 flex-1 overflow-auto bg-[radial-gradient(var(--border)_1px,transparent_1px)] [background-size:16px_16px] p-4"),
			h.Div(h.ID("frame"), h.Class("mx-auto h-full min-h-[480px] overflow-hidden rounded-xl border bg-background shadow-sm transition-[width] duration-200"),
				h.Iframe(h.ID("preview"), h.Title("Canvas"), h.Src("preview.html"), h.Class("size-full")),
			),
			h.Div(h.ID("error"), h.Hidden(), h.Class("absolute inset-x-4 bottom-4 max-h-48 overflow-auto rounded-lg border border-destructive/50 bg-background p-3 font-mono text-xs whitespace-pre-wrap text-destructive shadow-lg")),
			h.Div(h.ID("loading"), h.Class("absolute inset-0 flex flex-col items-center justify-center gap-3 bg-background/80 text-sm backdrop-blur-sm"),
				lucide.LoaderCircle(h.Class("size-5 animate-spin text-muted-foreground")),
				h.Div(h.ID("loading-text"), "Loading the Go engine…"),
				h.Div(h.Class("h-1 w-48 overflow-hidden rounded-full bg-muted"),
					h.Div(h.ID("loading-bar"), h.Class("h-full w-0 bg-primary transition-[width]")),
				),
			),
		),
	)
}

func rightPanel() h.Node {
	return h.Aside(h.Class("hidden min-h-0 flex-col border-l lg:flex"), h.ID("right"),
		panelTabs("design", []h.Node{
			tabs.Trigger(tabs.Value("design"), lucide.SlidersHorizontal(), "Design"),
			tabs.Trigger(tabs.Value("code"), lucide.Code(), "Code"),
		},
			panel{"design", "min-h-0 flex-1 overflow-auto",
				h.Div(h.ID("inspector"),
					h.Div(h.Class("flex flex-col items-center gap-2 p-8 text-center text-sm text-muted-foreground"),
						lucide.MousePointer2(h.Class("size-5")),
						"Select an element on the canvas or in Layers.",
					),
				),
			},
			panel{"code", "min-h-0 flex-1 overflow-hidden",
				h.Div(h.ID("editor"), h.Class("h-full")),
			},
		),
	)
}

// protos are the inspector's controls, cloned by studio.js so that they
// look like the rest of the page.
func protos() h.Node {
	proto := func(name string, n h.Node) h.Node {
		return h.Template(h.ID("proto-"+name), n)
	}
	action := func(name, label string, icon func(...any) *h.Element) h.Node {
		return button.Button(button.VariantGhost, button.SizeIconSm, h.Data("action", name), h.Aria("label", label), h.Title(label), icon())
	}
	return h.Group(
		proto("input", input.Input(h.Class("h-8 text-xs"))),
		proto("textarea", textarea.Textarea(h.Class("min-h-16 font-mono text-xs"), h.Attr("spellcheck", "false"))),
		proto("select", nativeselect.NativeSelect(h.Class("h-8 text-xs"))),
		proto("actions", h.Div(h.Class("flex items-center gap-0.5"),
			action("up", "Move up (Alt+↑)", lucide.ArrowUp),
			action("down", "Move down (Alt+↓)", lucide.ArrowDown),
			action("duplicate", "Duplicate (⌘D)", lucide.Copy),
			action("code", "Show in code", lucide.Code),
			action("delete", "Delete (⌫)", lucide.Trash2),
		)),
		proto("chip", button.Button(button.VariantOutline, button.SizeXs, h.Class("max-w-full justify-start font-mono"))),
		proto("label", label.Label(h.Class("text-xs text-muted-foreground"))),
		proto("badge", badge.Badge(badge.VariantSecondary, h.Class("font-mono"))),
	)
}

func statusBar() h.Node {
	return h.Footer(h.Class("flex h-7 shrink-0 items-center gap-3 border-t px-3 text-xs text-muted-foreground"),
		h.Span(h.ID("status"), "Loading…"),
		h.Div(h.Class("flex-1")),
		h.Span(h.Class("hidden sm:inline"), "Go in WebAssembly · yaegi · shadcn.go"),
	)
}

func itoa(i int) string {
	const digits = "0123456789"
	if i < 10 {
		return digits[i : i+1]
	}
	return itoa(i/10) + digits[i%10:i%10+1]
}
