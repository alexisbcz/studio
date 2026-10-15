package engine

// Snippet is an element of the visual editor's palette: an expression to
// insert, and the imports it needs.
type Snippet struct {
	Group   string   `json:"group"`
	Label   string   `json:"label"`
	Code    string   `json:"code"`
	Imports []Import `json:"imports"`
}

const (
	ui      = "github.com/alexisbcz/components/ui/"
	pLib    = "github.com/alexisbcz/components/lib"
	pLucide = "github.com/alexisbcz/icons/lucide"
)

func imp(paths ...string) []Import {
	out := []Import{{Name: "h", Path: libhtmlPath}}
	for _, p := range paths {
		name := ""
		switch p {
		case "dropdown-menu":
			name = "dm"
		case "toggle-group":
			name = "tg"
		case "radio-group":
			name = "rg"
		case "input-group":
			name = "ig"
		}
		path := p
		if p != pLib && p != pLucide {
			path = ui + p
		}
		out = append(out, Import{Name: name, Path: path})
	}
	return out
}

// Palette lists the snippets of the visual editor, by group.
var Palette = []Snippet{
	// Layout
	{"Layout", "Row", `h.Div(h.Class("flex items-center gap-2"))`, imp()},
	{"Layout", "Column", `h.Div(h.Class("flex flex-col gap-4"))`, imp()},
	{"Layout", "Grid", `h.Div(h.Class("grid grid-cols-2 gap-4"))`, imp()},
	{"Layout", "Centered page", `h.Div(h.Class("flex min-h-svh items-center justify-center p-6"))`, imp()},
	{"Layout", "Card", `card.Card(h.Class("w-full max-w-sm"),
	card.Header(
		card.Title("Card title"),
		card.Description("A short description of the card."),
	),
	card.Content(h.P("Card content")),
	card.Footer(button.Button("Action")),
)`, imp("card", "button")},
	{"Layout", "Separator", `separator.Separator()`, imp("separator")},
	{"Layout", "Tabs", `tabs.Tabs(tabs.Value("one"),
	tabs.List(
		tabs.Trigger(tabs.Value("one"), "One"),
		tabs.Trigger(tabs.Value("two"), "Two"),
	),
	tabs.Content(tabs.Value("one"), h.P("First tab")),
	tabs.Content(tabs.Value("two"), h.P("Second tab")),
)`, imp("tabs")},
	{"Layout", "Accordion", `accordion.Accordion(accordion.TypeSingle, accordion.Collapsible(true), h.Class("w-full"),
	accordion.Item(accordion.Value("item-1"),
		accordion.Trigger("Is it accessible?"),
		accordion.Content(h.P("Yes. It adheres to the WAI-ARIA design pattern.")),
	),
	accordion.Item(accordion.Value("item-2"),
		accordion.Trigger("Is it styled?"),
		accordion.Content(h.P("Yes. It comes with default styles.")),
	),
)`, imp("accordion")},
	{"Layout", "Scroll area", `scrollarea.ScrollArea(h.Class("h-48 w-48 rounded-md border"),
	h.Div(h.Class("p-4 text-sm"), "Scrollable content"),
)`, imp("scroll-area")},

	// Text
	{"Text", "Heading", `h.H2(h.Class("text-2xl font-semibold tracking-tight"), "Heading")`, imp()},
	{"Text", "Paragraph", `h.P(h.Class("leading-7 text-muted-foreground"), "Some text.")`, imp()},
	{"Text", "Link", `h.A(h.Href("#"), h.Class("font-medium underline underline-offset-4"), "Link")`, imp()},
	{"Text", "Badge", `badge.Badge("Badge")`, imp("badge")},
	{"Text", "Kbd", `kbd.Group(kbd.Kbd("⌘"), kbd.Kbd("K"))`, imp("kbd")},

	// Buttons
	{"Buttons", "Button", `button.Button("Button")`, imp("button")},
	{"Buttons", "Outline button", `button.Button(button.VariantOutline, "Button")`, imp("button")},
	{"Buttons", "Icon button", `button.Button(button.VariantOutline, button.SizeIcon, h.Aria("label", "Add"), lucide.Plus())`, imp("button", pLucide)},
	{"Buttons", "Button group", `buttongroup.ButtonGroup(
	button.Button(button.VariantOutline, "Archive"),
	button.Button(button.VariantOutline, "Report"),
)`, imp("button-group", "button")},
	{"Buttons", "Toggle", `toggle.Toggle(h.Aria("label", "Toggle bold"), lucide.Bold())`, imp("toggle", pLucide)},
	{"Buttons", "Toggle group", `tg.ToggleGroup(tg.VariantOutline, tg.TypeMultiple,
	tg.Item(tg.Value("bold"), h.Aria("label", "Bold"), lucide.Bold()),
	tg.Item(tg.Value("italic"), h.Aria("label", "Italic"), lucide.Italic()),
)`, imp("toggle-group", pLucide)},

	// Forms
	{"Forms", "Input", `input.Input(h.Type("email"), h.Placeholder("Email"))`, imp("input")},
	{"Forms", "Labeled input", `h.Div(h.Class("grid w-full max-w-sm gap-2"),
	label.Label(h.For("email"), "Email"),
	input.Input(h.ID("email"), h.Type("email"), h.Placeholder("m@example.com")),
)`, imp("label", "input")},
	{"Forms", "Textarea", `textarea.Textarea(h.Placeholder("Type your message here."))`, imp("textarea")},
	{"Forms", "Checkbox", `h.Div(h.Class("flex items-center gap-3"),
	checkbox.Checkbox(h.ID("terms")),
	label.Label(h.For("terms"), "Accept terms and conditions"),
)`, imp("checkbox", "label")},
	{"Forms", "Switch", `h.Div(h.Class("flex items-center gap-2"),
	switchui.Switch(h.ID("airplane-mode")),
	label.Label(h.For("airplane-mode"), "Airplane mode"),
)`, imp("switch", "label")},
	{"Forms", "Radio group", `rg.RadioGroup(rg.Value("one"),
	h.Div(h.Class("flex items-center gap-3"), rg.Item(rg.Value("one"), h.ID("r1")), label.Label(h.For("r1"), "One")),
	h.Div(h.Class("flex items-center gap-3"), rg.Item(rg.Value("two"), h.ID("r2")), label.Label(h.For("r2"), "Two")),
)`, imp("radio-group", "label")},
	{"Forms", "Select", `selectui.Select(
	selectui.Trigger(h.Class("w-[180px]"), selectui.Value(selectui.Placeholder("Select a fruit"))),
	selectui.Content(
		selectui.Item(selectui.ItemValue("apple"), "Apple"),
		selectui.Item(selectui.ItemValue("banana"), "Banana"),
	),
)`, imp("select")},
	{"Forms", "Native select", `nativeselect.NativeSelect(
	nativeselect.Option(h.Value("todo"), "Todo"),
	nativeselect.Option(h.Value("done"), "Done"),
)`, imp("native-select")},
	{"Forms", "Slider", `slider.Slider(slider.DefaultValue(50), slider.Max(100), slider.Step(1), h.Class("w-[60%]"))`, imp("slider")},
	{"Forms", "Input group", `ig.InputGroup(
	ig.Input(h.Placeholder("Search...")),
	ig.Addon(lucide.Search()),
)`, imp("input-group", pLucide)},
	{"Forms", "Field", `field.Field(
	field.Label(h.For("name"), "Name"),
	input.Input(h.ID("name"), h.Placeholder("Evil Rabbit")),
	field.Description("Your full name."),
)`, imp("field", "input")},

	// Display
	{"Display", "Alert", `alert.Alert(
	lucide.CircleCheck(),
	alert.Title("Success! Your changes have been saved"),
	alert.Description("This is an alert with icon, title and description."),
)`, imp("alert", pLucide)},
	{"Display", "Avatar", `avatar.Avatar(
	avatar.Image(h.Src("https://github.com/shadcn.png"), h.Alt("@shadcn")),
	avatar.Fallback("CN"),
)`, imp("avatar")},
	{"Display", "Progress", `progress.Progress(progress.Value(60), h.Class("w-[60%]"))`, imp("progress")},
	{"Display", "Skeleton", `skeleton.Skeleton(h.Class("h-4 w-[250px]"))`, imp("skeleton")},
	{"Display", "Spinner", `spinner.Spinner()`, imp("spinner")},
	{"Display", "Item", `item.Item(item.VariantOutline,
	item.Content(
		item.Title("Basic Item"),
		item.Description("A simple item with title and description."),
	),
	item.Actions(button.Button(button.VariantOutline, button.SizeSm, "Action")),
)`, imp("item", "button")},
	{"Display", "Empty state", `empty.Empty(
	empty.Header(
		empty.Media(empty.MediaIcon, lucide.FolderCode()),
		empty.Title("No projects yet"),
		empty.Description("Get started by creating your first project."),
	),
	empty.Content(button.Button("Create project")),
)`, imp("empty", "button", pLucide)},
	{"Display", "Table", `table.Table(
	table.Header(table.Row(table.Head("Invoice"), table.Head("Status"), table.Head(h.Class("text-right"), "Amount"))),
	table.Body(
		table.Row(table.Cell(h.Class("font-medium"), "INV001"), table.Cell("Paid"), table.Cell(h.Class("text-right"), "$250.00")),
		table.Row(table.Cell(h.Class("font-medium"), "INV002"), table.Cell("Pending"), table.Cell(h.Class("text-right"), "$150.00")),
	),
)`, imp("table")},
	{"Display", "Calendar", `calendar.Calendar(calendar.ModeSingle, h.Class("rounded-md border shadow-sm"))`, imp("calendar")},
	{"Display", "Icon", `lucide.Sparkles(h.Class("size-4"))`, imp(pLucide)},

	// Overlays
	{"Overlays", "Dialog", `dialog.Dialog(
	dialog.Trigger(lib.AsChild, button.Button(button.VariantOutline, "Open dialog")),
	dialog.Content(h.Class("sm:max-w-[425px]"),
		dialog.Header(
			dialog.Title("Edit profile"),
			dialog.Description("Make changes to your profile here."),
		),
		dialog.Footer(
			dialog.Close(lib.AsChild, button.Button(button.VariantOutline, "Cancel")),
			button.Button("Save changes"),
		),
	),
)`, imp("dialog", "button", pLib)},
	{"Overlays", "Alert dialog", `alertdialog.AlertDialog(
	alertdialog.Trigger(lib.AsChild, button.Button(button.VariantOutline, "Delete")),
	alertdialog.Content(
		alertdialog.Header(
			alertdialog.Title("Are you absolutely sure?"),
			alertdialog.Description("This action cannot be undone."),
		),
		alertdialog.Footer(alertdialog.Cancel("Cancel"), alertdialog.Action("Continue")),
	),
)`, imp("alert-dialog", "button", pLib)},
	{"Overlays", "Sheet", `sheet.Sheet(
	sheet.Trigger(lib.AsChild, button.Button(button.VariantOutline, "Open sheet")),
	sheet.Content(
		sheet.Header(
			sheet.Title("Edit profile"),
			sheet.Description("Make changes to your profile here."),
		),
	),
)`, imp("sheet", "button", pLib)},
	{"Overlays", "Popover", `popover.Popover(
	popover.Trigger(lib.AsChild, button.Button(button.VariantOutline, "Open popover")),
	popover.Content(h.Class("w-80"), h.P(h.Class("text-sm"), "Popover content")),
)`, imp("popover", "button", pLib)},
	{"Overlays", "Tooltip", `tooltip.Tooltip(
	tooltip.Trigger(lib.AsChild, button.Button(button.VariantOutline, "Hover")),
	tooltip.Content(h.P("Add to library")),
)`, imp("tooltip", "button", pLib)},
	{"Overlays", "Dropdown menu", `dm.DropdownMenu(
	dm.Trigger(lib.AsChild, button.Button(button.VariantOutline, "Open")),
	dm.Content(h.Class("w-56"), dm.AlignStart,
		dm.Label("My Account"),
		dm.Group(
			dm.Item("Profile", dm.Shortcut("⇧⌘P")),
			dm.Item("Settings", dm.Shortcut("⌘S")),
		),
		dm.Separator(),
		dm.Item("Log out"),
	),
)`, imp("dropdown-menu", "button", pLib)},
	{"Overlays", "Hover card", `hovercard.HoverCard(
	hovercard.Trigger(lib.AsChild, button.Button(button.VariantLink, "@nextjs")),
	hovercard.Content(h.Class("w-80"), h.P(h.Class("text-sm"), "The React Framework.")),
)`, imp("hover-card", "button", pLib)},

	// Navigation
	{"Navigation", "Breadcrumb", `breadcrumb.Breadcrumb(breadcrumb.List(
	breadcrumb.Item(breadcrumb.Link(h.Href("#"), "Home")),
	breadcrumb.Separator(),
	breadcrumb.Item(breadcrumb.Page("Breadcrumb")),
))`, imp("breadcrumb")},
	{"Navigation", "Collapsible", `collapsible.Collapsible(h.Class("flex w-[350px] flex-col gap-2"),
	collapsible.Trigger(lib.AsChild, button.Button(button.VariantGhost, "Toggle")),
	collapsible.Content(h.Div(h.Class("rounded-md border px-4 py-2 text-sm"), "Hidden content")),
)`, imp("collapsible", "button", pLib)},
}
