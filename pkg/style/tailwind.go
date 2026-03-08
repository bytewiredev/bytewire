// Package style provides type-safe CSS generation for Bytewire components.
// Styling rules are Go constants — the compiler catches invalid styles at build time.
package style

// Class represents a CSS utility class name.
type Class string

// Common layout classes
const (
	Flex       Class = "flex"
	Flex1      Class = "flex-1"
	FlexCol    Class = "flex-col"
	FlexRow    Class = "flex-row"
	FlexWrap   Class = "flex-wrap"
	Grid       Class = "grid"
	ItemsCenter Class = "items-center"
	JustifyCenter Class = "justify-center"
	JustifyBetween Class = "justify-between"
	Gap1       Class = "gap-1"
	Gap2       Class = "gap-2"
	Gap4       Class = "gap-4"
	Gap8       Class = "gap-8"
)

// Spacing
const (
	P1     Class = "p-1"
	P2     Class = "p-2"
	P4     Class = "p-4"
	P8     Class = "p-8"
	M1     Class = "m-1"
	M2     Class = "m-2"
	M4     Class = "m-4"
	M8     Class = "m-8"
	Mx2    Class = "mx-2"
	MxAuto Class = "mx-auto"
	My2    Class = "my-2"
	Px4    Class = "px-4"
	Py2    Class = "py-2"
)

// Typography
const (
	TextSm   Class = "text-sm"
	TextBase Class = "text-base"
	TextLg   Class = "text-lg"
	TextXl   Class = "text-xl"
	Text2Xl  Class = "text-2xl"
	Text3Xl  Class = "text-3xl"
	FontBold Class = "font-bold"
	FontMedium Class = "font-medium"
	TextCenter Class = "text-center"
)

// Colors
const (
	BgWhite    Class = "bg-white"
	BgGray100  Class = "bg-gray-100"
	BgGray800  Class = "bg-gray-800"
	BgBlue500  Class = "bg-blue-500"
	BgBlue600  Class = "bg-blue-600"
	BgGreen500 Class = "bg-green-500"
	BgRed500   Class = "bg-red-500"
	TextWhite  Class = "text-white"
	TextGray700 Class = "text-gray-700"
	TextGray900 Class = "text-gray-900"
	TextBlue500 Class = "text-blue-500"
)

// Borders & Rounding
const (
	Rounded    Class = "rounded"
	RoundedMd  Class = "rounded-md"
	RoundedLg  Class = "rounded-lg"
	RoundedFull Class = "rounded-full"
	Border     Class = "border"
	BorderGray300 Class = "border-gray-300"
	Shadow     Class = "shadow"
	ShadowMd   Class = "shadow-md"
	ShadowLg   Class = "shadow-lg"
)

// Sizing
const (
	WFull   Class = "w-full"
	HFull   Class = "h-full"
	WScreen Class = "w-screen"
	HScreen Class = "h-screen"
	W4      Class = "w-4"
	H4      Class = "h-4"
	W6      Class = "w-6"
	H6      Class = "h-6"
	W8      Class = "w-8"
	H8      Class = "h-8"
	MaxWLg  Class = "max-w-lg"
	MaxWXl  Class = "max-w-xl"
	MaxW2Xl Class = "max-w-2xl"
	MaxW3Xl Class = "max-w-3xl"
	MaxW4Xl Class = "max-w-4xl"
	MinW0   Class = "min-w-0"
)

// Positioning
const (
	Fixed    Class = "fixed"
	Absolute Class = "absolute"
	Relative Class = "relative"
	Inset0   Class = "inset-0"
	Top0     Class = "top-0"
	Right0   Class = "right-0"
)

// Z-Index
const (
	Z10 Class = "z-10"
	Z50 Class = "z-50"
)

// Display & Overflow
const (
	Hidden    Class = "hidden"
	Block     Class = "block"
	Inline    Class = "inline-block"
	Overflow  Class = "overflow-auto"
	BoxBorder Class = "box-border"
)

// Cursor
const (
	CursorPointer Class = "cursor-pointer"
)

// Additional Colors
const (
	BgBlack       Class = "bg-black"
	BgYellow500   Class = "bg-yellow-500"
	BgYellow100   Class = "bg-yellow-100"
	BgGreen100    Class = "bg-green-100"
	BgRed100      Class = "bg-red-100"
	BgBlue100     Class = "bg-blue-100"
	BgOpacity50   Class = "bg-opacity-50"
	TextYellow800 Class = "text-yellow-800"
	TextGreen800  Class = "text-green-800"
	TextRed800    Class = "text-red-800"
	TextRed500    Class = "text-red-500"
	TextGray500   Class = "text-gray-500"
	TextGray600   Class = "text-gray-600"
)

// Additional Spacing
const (
	P0  Class = "p-0"
	P3  Class = "p-3"
	P6  Class = "p-6"
	Px2 Class = "px-2"
	Px3 Class = "px-3"
	Py1 Class = "py-1"
	Py3 Class = "py-3"
	Mt2 Class = "mt-2"
	Mb2 Class = "mb-2"
	Mb4 Class = "mb-4"
	Ml2 Class = "ml-2"
	Mr2 Class = "mr-2"
)

// Additional Borders
const (
	BorderL4         Class = "border-l-4"
	BorderGreen500   Class = "border-green-500"
	BorderRed500     Class = "border-red-500"
	BorderYellow500  Class = "border-yellow-500"
	BorderBlue500    Class = "border-blue-500"
)

// Animation
const (
	AnimateSpin Class = "animate-spin"
)

// Sidebar layout
const (
	W64          Class = "w-64"
	MinHScreen   Class = "min-h-screen"
	Shrink0      Class = "shrink-0"
	OverflowYAuto Class = "overflow-y-auto"
)

// Grid
const (
	GridCols2 Class = "grid-cols-2"
	GridCols3 Class = "grid-cols-3"
	GridCols4 Class = "grid-cols-4"
	ColSpan2  Class = "col-span-2"
)

// Additional colors for showcase
const (
	BgGray50   Class = "bg-gray-50"
	BgGray900  Class = "bg-gray-900"
	TextGray300 Class = "text-gray-300"
	TextGray400 Class = "text-gray-400"
)

// Additional spacing
const (
	Px6  Class = "px-6"
	Py8  Class = "py-8"
	Mb6  Class = "mb-6"
	Gap6 Class = "gap-6"
)

// Additional typography
const (
	TextXs       Class = "text-xs"
	Uppercase    Class = "uppercase"
	TrackingWide Class = "tracking-wide"
)

// Additional borders
const (
	BorderB Class = "border-b"
)

// Classes joins multiple type-safe classes into a single string.
func Classes(classes ...Class) string {
	if len(classes) == 0 {
		return ""
	}
	result := string(classes[0])
	for _, c := range classes[1:] {
		result += " " + string(c)
	}
	return result
}
