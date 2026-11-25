package main

func marginBoxStylePrefixes() []boxStylePrefix {
	return []boxStylePrefix{
		{prefix: "m", autoAllowed: true, fields: []string{"margin.top", "margin.bottom", "margin.left", "margin.right"}, docStringPrefix: "Sets the margin of the element. [Docs](https://tailwindcss.com/docs/margin)"},
		{prefix: "mt", autoAllowed: true, fields: []string{"margin.top"}, docStringPrefix: "Sets the top margin of the element. [Docs](https://tailwindcss.com/docs/margin#add-margin-to-a-single-side)"},
		{prefix: "mb", autoAllowed: true, fields: []string{"margin.bottom"}, docStringPrefix: "Sets the bottom margin of the element. [Docs](https://tailwindcss.com/docs/margin#add-margin-to-a-single-side)"},
		{prefix: "my", autoAllowed: true, fields: []string{"margin.top", "margin.bottom"}, docStringPrefix: "Sets the vertical margin of the element. [Docs](https://tailwindcss.com/docs/margin#add-vertical-margin)"},
		{prefix: "mx", autoAllowed: true, fields: []string{"margin.left", "margin.right"}, docStringPrefix: "Sets the horizontal margin of the element. [Docs](https://tailwindcss.com/docs/margin#add-horizontal-margin)"},
		{prefix: "ml", autoAllowed: true, fields: []string{"margin.left"}, docStringPrefix: "Sets the left margin of the element. [Docs](https://tailwindcss.com/docs/margin#add-margin-to-a-single-side)"},
		{prefix: "mr", autoAllowed: true, fields: []string{"margin.right"}, docStringPrefix: "Sets the right margin of the element. [Docs](https://tailwindcss.com/docs/margin#add-margin-to-a-single-side)"},
	}
}

func paddingBoxStylePrefixes() []boxStylePrefix {
	return []boxStylePrefix{
		{prefix: "p", autoAllowed: false, fields: []string{"padding.top", "padding.bottom", "padding.left", "padding.right"}, docStringPrefix: "Sets the padding of the element. [Docs](https://tailwindcss.com/docs/padding)"},
		{prefix: "pt", autoAllowed: false, fields: []string{"padding.top"}, docStringPrefix: "Sets the top padding of the element. [Docs](https://tailwindcss.com/docs/padding#add-padding-to-a-single-side)"},
		{prefix: "pb", autoAllowed: false, fields: []string{"padding.bottom"}, docStringPrefix: "Sets the bottom padding of the element. [Docs](https://tailwindcss.com/docs/padding#add-padding-to-a-single-side)"},
		{prefix: "px", autoAllowed: false, fields: []string{"padding.left", "padding.right"}, docStringPrefix: "Sets the horizontal padding of the element. [Docs](https://tailwindcss.com/docs/padding#add-horizontal-padding)"},
		{prefix: "py", autoAllowed: false, fields: []string{"padding.top", "padding.bottom"}, docStringPrefix: "Sets the vertical padding of the element. [Docs](https://tailwindcss.com/docs/padding#add-vertical-padding)"},
		{prefix: "pl", autoAllowed: false, fields: []string{"padding.left"}, docStringPrefix: "Sets the left padding of the element. [Docs](https://tailwindcss.com/docs/padding#add-padding-to-a-single-side)"},
		{prefix: "pr", autoAllowed: false, fields: []string{"padding.right"}, docStringPrefix: "Sets the right padding of the element. [Docs](https://tailwindcss.com/docs/padding#add-padding-to-a-single-side)"},
	}
}

func positionBoxStylePrefixes() []boxStylePrefix {
	return []boxStylePrefix{
		{prefix: "inset", autoAllowed: true, fields: []string{"inset.top", "inset.right", "inset.bottom", "inset.left"}, docStringPrefix: "Sets the top, right, bottom, and left values of a positioned element. [Docs](https://tailwindcss.com/docs/top-right-bottom-left)"},
		{prefix: "top", autoAllowed: true, fields: []string{"inset.top"}, docStringPrefix: "Sets the top value of a positioned element. [Docs](https://tailwindcss.com/docs/top-right-bottom-left)"},
		{prefix: "bottom", autoAllowed: true, fields: []string{"inset.bottom"}, docStringPrefix: "Sets the bottom value of a positioned element. [Docs](https://tailwindcss.com/docs/top-right-bottom-left)"},
		{prefix: "left", autoAllowed: true, fields: []string{"inset.left"}, docStringPrefix: "Sets the left value of a positioned element. [Docs](https://tailwindcss.com/docs/top-right-bottom-left)"},
		{prefix: "right", autoAllowed: true, fields: []string{"inset.right"}, docStringPrefix: "Sets the right value of a positioned element. [Docs](https://tailwindcss.com/docs/top-right-bottom-left)"},
	}
}

func boxPrefixes() []boxStylePrefix {
	return []boxStylePrefix{
		{prefix: "w", autoAllowed: true, fields: []string{"size.width"}, docStringPrefix: "Sets the width of the element. [Docs](https://tailwindcss.com/docs/width)"},
		{prefix: "h", autoAllowed: true, fields: []string{"size.height"}, docStringPrefix: "Sets the height of the element. [Docs](https://tailwindcss.com/docs/height)"},
		{prefix: "size", autoAllowed: true, fields: []string{"size.width", "size.height"}, docStringPrefix: "Sets the width and height of the element."},
		{prefix: "min_size", autoAllowed: true, fields: []string{"min_size.width", "min_size.height"}, docStringPrefix: "Sets the minimum width and height of the element."},
		{prefix: "min_w", autoAllowed: true, fields: []string{"min_size.width"}, docStringPrefix: "Sets the minimum width of the element. [Docs](https://tailwindcss.com/docs/min-width)"},
		{prefix: "min_h", autoAllowed: true, fields: []string{"min_size.height"}, docStringPrefix: "Sets the minimum height of the element. [Docs](https://tailwindcss.com/docs/min-height)"},
		{prefix: "max_size", autoAllowed: true, fields: []string{"max_size.width", "max_size.height"}, docStringPrefix: "Sets the maximum width and height of the element."},
		{prefix: "max_w", autoAllowed: true, fields: []string{"max_size.width"}, docStringPrefix: "Sets the maximum width of the element. [Docs](https://tailwindcss.com/docs/max-width)"},
		{prefix: "max_h", autoAllowed: true, fields: []string{"max_size.height"}, docStringPrefix: "Sets the maximum height of the element. [Docs](https://tailwindcss.com/docs/max-height)"},
		{prefix: "gap", autoAllowed: false, fields: []string{"gap.width", "gap.height"}, docStringPrefix: "Sets the gap between rows and columns in flex layouts. [Docs](https://tailwindcss.com/docs/gap)"},
		{prefix: "gap_x", autoAllowed: false, fields: []string{"gap.width"}, docStringPrefix: "Sets the gap between columns in flex layouts. [Docs](https://tailwindcss.com/docs/gap#changing-row-and-column-gaps-independently)"},
		{prefix: "gap_y", autoAllowed: false, fields: []string{"gap.height"}, docStringPrefix: "Sets the gap between rows in flex layouts. [Docs](https://tailwindcss.com/docs/gap#changing-row-and-column-gaps-independently)"},
	}
}

func boxStyleSuffixes() []boxStyleSuffix {
	return []boxStyleSuffix{
		{suffix: "0", tokens: lengthTokens{fn: "px", arg: "0."}, docStringSuffix: "0px"},
		{suffix: "0p5", tokens: lengthTokens{fn: "rems", arg: "0.125"}, docStringSuffix: "2px (0.125rem)"},
		{suffix: "1", tokens: lengthTokens{fn: "rems", arg: "0.25"}, docStringSuffix: "4px (0.25rem)"},
		{suffix: "1p5", tokens: lengthTokens{fn: "rems", arg: "0.375"}, docStringSuffix: "6px (0.375rem)"},
		{suffix: "2", tokens: lengthTokens{fn: "rems", arg: "0.5"}, docStringSuffix: "8px (0.5rem)"},
		{suffix: "2p5", tokens: lengthTokens{fn: "rems", arg: "0.625"}, docStringSuffix: "10px (0.625rem)"},
		{suffix: "3", tokens: lengthTokens{fn: "rems", arg: "0.75"}, docStringSuffix: "12px (0.75rem)"},
		{suffix: "3p5", tokens: lengthTokens{fn: "rems", arg: "0.875"}, docStringSuffix: "14px (0.875rem)"},
		{suffix: "4", tokens: lengthTokens{fn: "rems", arg: "1."}, docStringSuffix: "16px (1rem)"},
		{suffix: "5", tokens: lengthTokens{fn: "rems", arg: "1.25"}, docStringSuffix: "20px (1.25rem)"},
		{suffix: "6", tokens: lengthTokens{fn: "rems", arg: "1.5"}, docStringSuffix: "24px (1.5rem)"},
		{suffix: "7", tokens: lengthTokens{fn: "rems", arg: "1.75"}, docStringSuffix: "28px (1.75rem)"},
		{suffix: "8", tokens: lengthTokens{fn: "rems", arg: "2.0"}, docStringSuffix: "32px (2rem)"},
		{suffix: "9", tokens: lengthTokens{fn: "rems", arg: "2.25"}, docStringSuffix: "36px (2.25rem)"},
		{suffix: "10", tokens: lengthTokens{fn: "rems", arg: "2.5"}, docStringSuffix: "40px (2.5rem)"},
		{suffix: "11", tokens: lengthTokens{fn: "rems", arg: "2.75"}, docStringSuffix: "44px (2.75rem)"},
		{suffix: "12", tokens: lengthTokens{fn: "rems", arg: "3."}, docStringSuffix: "48px (3rem)"},
		{suffix: "16", tokens: lengthTokens{fn: "rems", arg: "4."}, docStringSuffix: "64px (4rem)"},
		{suffix: "20", tokens: lengthTokens{fn: "rems", arg: "5."}, docStringSuffix: "80px (5rem)"},
		{suffix: "24", tokens: lengthTokens{fn: "rems", arg: "6."}, docStringSuffix: "96px (6rem)"},
		{suffix: "32", tokens: lengthTokens{fn: "rems", arg: "8."}, docStringSuffix: "128px (8rem)"},
		{suffix: "40", tokens: lengthTokens{fn: "rems", arg: "10."}, docStringSuffix: "160px (10rem)"},
		{suffix: "48", tokens: lengthTokens{fn: "rems", arg: "12."}, docStringSuffix: "192px (12rem)"},
		{suffix: "56", tokens: lengthTokens{fn: "rems", arg: "14."}, docStringSuffix: "224px (14rem)"},
		{suffix: "64", tokens: lengthTokens{fn: "rems", arg: "16."}, docStringSuffix: "256px (16rem)"},
		{suffix: "72", tokens: lengthTokens{fn: "rems", arg: "18."}, docStringSuffix: "288px (18rem)"},
		{suffix: "80", tokens: lengthTokens{fn: "rems", arg: "20."}, docStringSuffix: "320px (20rem)"},
		{suffix: "96", tokens: lengthTokens{fn: "rems", arg: "24."}, docStringSuffix: "384px (24rem)"},
		{suffix: "112", tokens: lengthTokens{fn: "rems", arg: "28."}, docStringSuffix: "448px (28rem)"},
		{suffix: "128", tokens: lengthTokens{fn: "rems", arg: "32."}, docStringSuffix: "512px (32rem)"},
		{suffix: "auto", tokens: lengthTokens{fn: "auto", arg: ""}, docStringSuffix: "Auto"},
		{suffix: "px", tokens: lengthTokens{fn: "px", arg: "1."}, docStringSuffix: "1px"},
		{suffix: "full", tokens: lengthTokens{fn: "relative", arg: "1."}, docStringSuffix: "100%"},
		{suffix: "1_2", tokens: lengthTokens{fn: "relative", arg: "0.5"}, docStringSuffix: "50% (1/2)"},
		{suffix: "1_3", tokens: lengthTokens{fn: "relative", arg: "1./3."}, docStringSuffix: "33% (1/3)"},
		{suffix: "2_3", tokens: lengthTokens{fn: "relative", arg: "2./3."}, docStringSuffix: "66% (2/3)"},
		{suffix: "1_4", tokens: lengthTokens{fn: "relative", arg: "0.25"}, docStringSuffix: "25% (1/4)"},
		{suffix: "2_4", tokens: lengthTokens{fn: "relative", arg: "0.5"}, docStringSuffix: "50% (2/4)"},
		{suffix: "3_4", tokens: lengthTokens{fn: "relative", arg: "0.75"}, docStringSuffix: "75% (3/4)"},
		{suffix: "1_5", tokens: lengthTokens{fn: "relative", arg: "0.2"}, docStringSuffix: "20% (1/5)"},
		{suffix: "2_5", tokens: lengthTokens{fn: "relative", arg: "0.4"}, docStringSuffix: "40% (2/5)"},
		{suffix: "3_5", tokens: lengthTokens{fn: "relative", arg: "0.6"}, docStringSuffix: "60% (3/5)"},
		{suffix: "4_5", tokens: lengthTokens{fn: "relative", arg: "0.8"}, docStringSuffix: "80% (4/5)"},
		{suffix: "1_6", tokens: lengthTokens{fn: "relative", arg: "1./6."}, docStringSuffix: "16% (1/6)"},
		{suffix: "5_6", tokens: lengthTokens{fn: "relative", arg: "5./6."}, docStringSuffix: "80% (5/6)"},
		{suffix: "1_12", tokens: lengthTokens{fn: "relative", arg: "1./12."}, docStringSuffix: "8% (1/12)"},
	}
}

func cornerPrefixes() []cornerStylePrefix {
	return []cornerStylePrefix{
		{prefix: "rounded", fields: []string{"corner_radii.top_left", "corner_radii.top_right", "corner_radii.bottom_right", "corner_radii.bottom_left"}, docStringPrefix: "Sets the border radius of the element. [Docs](https://tailwindcss.com/docs/border-radius)"},
		{prefix: "rounded_t", fields: []string{"corner_radii.top_left", "corner_radii.top_right"}, docStringPrefix: "Sets the border radius of the top side of the element. [Docs](https://tailwindcss.com/docs/border-radius#rounding-sides-separately)"},
		{prefix: "rounded_b", fields: []string{"corner_radii.bottom_left", "corner_radii.bottom_right"}, docStringPrefix: "Sets the border radius of the bottom side of the element. [Docs](https://tailwindcss.com/docs/border-radius#rounding-sides-separately)"},
		{prefix: "rounded_r", fields: []string{"corner_radii.top_right", "corner_radii.bottom_right"}, docStringPrefix: "Sets the border radius of the right side of the element. [Docs](https://tailwindcss.com/docs/border-radius#rounding-sides-separately)"},
		{prefix: "rounded_l", fields: []string{"corner_radii.top_left", "corner_radii.bottom_left"}, docStringPrefix: "Sets the border radius of the left side of the element. [Docs](https://tailwindcss.com/docs/border-radius#rounding-sides-separately)"},
		{prefix: "rounded_tl", fields: []string{"corner_radii.top_left"}, docStringPrefix: "Sets the border radius of the top left corner of the element. [Docs](https://tailwindcss.com/docs/border-radius#rounding-corners-separately)"},
		{prefix: "rounded_tr", fields: []string{"corner_radii.top_right"}, docStringPrefix: "Sets the border radius of the top right corner of the element. [Docs](https://tailwindcss.com/docs/border-radius#rounding-corners-separately)"},
		{prefix: "rounded_bl", fields: []string{"corner_radii.bottom_left"}, docStringPrefix: "Sets the border radius of the bottom left corner of the element. [Docs](https://tailwindcss.com/docs/border-radius#rounding-corners-separately)"},
		{prefix: "rounded_br", fields: []string{"corner_radii.bottom_right"}, docStringPrefix: "Sets the border radius of the bottom right corner of the element. [Docs](https://tailwindcss.com/docs/border-radius#rounding-corners-separately)"},
	}
}

func cornerSuffixes() []cornerStyleSuffix {
	return []cornerStyleSuffix{
		{suffix: "none", tokens: lengthTokens{fn: "px", arg: "0."}, docStringSuffix: "0px"},
		{suffix: "xs", tokens: lengthTokens{fn: "rems", arg: "0.125"}, docStringSuffix: "2px (0.125rem)"},
		{suffix: "sm", tokens: lengthTokens{fn: "rems", arg: "0.25"}, docStringSuffix: "4px (0.25rem)"},
		{suffix: "md", tokens: lengthTokens{fn: "rems", arg: "0.375"}, docStringSuffix: "6px (0.375rem)"},
		{suffix: "lg", tokens: lengthTokens{fn: "rems", arg: "0.5"}, docStringSuffix: "8px (0.5rem)"},
		{suffix: "xl", tokens: lengthTokens{fn: "rems", arg: "0.75"}, docStringSuffix: "12px (0.75rem)"},
		{suffix: "2xl", tokens: lengthTokens{fn: "rems", arg: "1."}, docStringSuffix: "16px (1rem)"},
		{suffix: "3xl", tokens: lengthTokens{fn: "rems", arg: "1.5"}, docStringSuffix: "24px (1.5rem)"},
		{suffix: "full", tokens: lengthTokens{fn: "px", arg: "9999."}, docStringSuffix: "9999px"},
	}
}

func borderPrefixes() []borderStylePrefix {
	return []borderStylePrefix{
		{prefix: "border", fields: []string{"border_widths.top", "border_widths.right", "border_widths.bottom", "border_widths.left"}, docStringPrefix: "Sets the border width of the element. [Docs](https://tailwindcss.com/docs/border-width)"},
		{prefix: "border_t", fields: []string{"border_widths.top"}, docStringPrefix: "Sets the border width of the top side of the element. [Docs](https://tailwindcss.com/docs/border-width#individual-sides)"},
		{prefix: "border_b", fields: []string{"border_widths.bottom"}, docStringPrefix: "Sets the border width of the bottom side of the element. [Docs](https://tailwindcss.com/docs/border-width#individual-sides)"},
		{prefix: "border_r", fields: []string{"border_widths.right"}, docStringPrefix: "Sets the border width of the right side of the element. [Docs](https://tailwindcss.com/docs/border-width#individual-sides)"},
		{prefix: "border_l", fields: []string{"border_widths.left"}, docStringPrefix: "Sets the border width of the left side of the element. [Docs](https://tailwindcss.com/docs/border-width#individual-sides)"},
		{prefix: "border_x", fields: []string{"border_widths.left", "border_widths.right"}, docStringPrefix: "Sets the border width of the vertical sides of the element. [Docs](https://tailwindcss.com/docs/border-width#horizontal-and-vertical-sides)"},
		{prefix: "border_y", fields: []string{"border_widths.top", "border_widths.bottom"}, docStringPrefix: "Sets the border width of the horizontal sides of the element. [Docs](https://tailwindcss.com/docs/border-width#horizontal-and-vertical-sides)"},
	}
}

func borderSuffixes() []borderStyleSuffix {
	return []borderStyleSuffix{
		{suffix: "0", tokens: lengthTokens{fn: "px", arg: "0."}, docStringSuffix: "0px"},
		{suffix: "1", tokens: lengthTokens{fn: "px", arg: "1."}, docStringSuffix: "1px"},
		{suffix: "2", tokens: lengthTokens{fn: "px", arg: "2."}, docStringSuffix: "2px"},
		{suffix: "3", tokens: lengthTokens{fn: "px", arg: "3."}, docStringSuffix: "3px"},
		{suffix: "4", tokens: lengthTokens{fn: "px", arg: "4."}, docStringSuffix: "4px"},
		{suffix: "5", tokens: lengthTokens{fn: "px", arg: "5."}, docStringSuffix: "5px"},
		{suffix: "6", tokens: lengthTokens{fn: "px", arg: "6."}, docStringSuffix: "6px"},
		{suffix: "7", tokens: lengthTokens{fn: "px", arg: "7."}, docStringSuffix: "7px"},
		{suffix: "8", tokens: lengthTokens{fn: "px", arg: "8."}, docStringSuffix: "8px"},
		{suffix: "9", tokens: lengthTokens{fn: "px", arg: "9."}, docStringSuffix: "9px"},
		{suffix: "10", tokens: lengthTokens{fn: "px", arg: "10."}, docStringSuffix: "10px"},
		{suffix: "11", tokens: lengthTokens{fn: "px", arg: "11."}, docStringSuffix: "11px"},
		{suffix: "12", tokens: lengthTokens{fn: "px", arg: "12."}, docStringSuffix: "12px"},
		{suffix: "16", tokens: lengthTokens{fn: "px", arg: "16."}, docStringSuffix: "16px"},
		{suffix: "20", tokens: lengthTokens{fn: "px", arg: "20."}, docStringSuffix: "20px"},
		{suffix: "24", tokens: lengthTokens{fn: "px", arg: "24."}, docStringSuffix: "24px"},
		{suffix: "32", tokens: lengthTokens{fn: "px", arg: "32."}, docStringSuffix: "32px"},
	}
}
