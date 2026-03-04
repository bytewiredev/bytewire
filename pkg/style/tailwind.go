// Package style provides type-safe CSS generation for CBS components.
// Styling rules are Go constants — the compiler catches invalid styles at build time.
package style

// Class represents a CSS utility class name.
type Class string

// Common layout classes
const (
	Flex       Class = "flex"
	FlexCol    Class = "flex-col"
	FlexRow    Class = "flex-row"
	FlexWrap   Class = "flex-wrap"
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
	P1  Class = "p-1"
	P2  Class = "p-2"
	P4  Class = "p-4"
	P8  Class = "p-8"
	M1  Class = "m-1"
	M2  Class = "m-2"
	M4  Class = "m-4"
	M8  Class = "m-8"
	Mx2 Class = "mx-2"
	My2 Class = "my-2"
	Px4 Class = "px-4"
	Py2 Class = "py-2"
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
	MaxWLg  Class = "max-w-lg"
	MaxWXl  Class = "max-w-xl"
	MaxW2Xl Class = "max-w-2xl"
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
