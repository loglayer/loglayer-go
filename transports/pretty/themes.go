package pretty

import "github.com/fatih/color"

// Style is a function that wraps a string in ANSI escape codes. The transport
// is theme-agnostic — supply any `func(string) string` that produces colored
// output (or no-op).
type Style func(string) string

// Theme holds the styles applied to each log level and to structured data
// rendered alongside the log line.
type Theme struct {
	Trace     Style
	Debug     Style
	Info      Style
	Warn      Style
	Error     Style
	Fatal     Style
	LogID     Style
	Timestamp Style
	DataKey   Style
	DataValue Style
}

// rgb returns a Style backed by 24-bit ANSI color.
func rgb(r, g, b int) Style {
	c := color.RGB(r, g, b)
	return func(s string) string { return c.Sprint(s) }
}

// rgbBgFg returns a Style with an RGB background and the given foreground RGB.
func rgbBgFg(bgR, bgG, bgB, fgR, fgG, fgB int) Style {
	c := color.BgRGB(bgR, bgG, bgB).AddRGB(fgR, fgG, fgB)
	return func(s string) string { return c.Sprint(s) }
}

// noStyle returns the input unchanged. Used when the user disables colors.
func noStyle(s string) string { return s }

// Moonlight is the default theme. Cool blues and soft greens; works best on a
// dark terminal.
func Moonlight() *Theme {
	return &Theme{
		Trace:     rgb(114, 135, 153),
		Debug:     rgb(130, 170, 255),
		Info:      rgb(195, 232, 141),
		Warn:      rgb(255, 203, 107),
		Error:     rgb(247, 118, 142),
		Fatal:     rgbBgFg(247, 118, 142, 255, 255, 255),
		LogID:     rgb(84, 98, 117),
		Timestamp: rgb(84, 98, 117),
		DataKey:   rgb(130, 170, 255),
		DataValue: rgb(209, 219, 231),
	}
}

// Sunlight is a light-terminal theme with deep, rich colors.
func Sunlight() *Theme {
	return &Theme{
		Trace:     rgb(110, 110, 110),
		Debug:     rgb(32, 96, 159),
		Info:      rgb(35, 134, 54),
		Warn:      rgb(176, 95, 0),
		Error:     rgb(191, 0, 0),
		Fatal:     rgbBgFg(191, 0, 0, 255, 255, 255),
		LogID:     rgb(110, 110, 110),
		Timestamp: rgb(110, 110, 110),
		DataKey:   rgb(32, 96, 159),
		DataValue: rgb(0, 0, 0),
	}
}

// Neon is a dark cyberpunk theme with vivid colors.
func Neon() *Theme {
	return &Theme{
		Trace:     rgb(108, 108, 255),
		Debug:     rgb(255, 82, 246),
		Info:      rgb(0, 255, 163),
		Warn:      rgb(255, 231, 46),
		Error:     rgb(255, 53, 91),
		Fatal:     rgbBgFg(255, 53, 91, 0, 255, 163),
		LogID:     rgb(187, 134, 252),
		Timestamp: rgb(187, 134, 252),
		DataKey:   rgb(0, 255, 240),
		DataValue: rgb(255, 255, 255),
	}
}

// Nature is a light-terminal theme with earthy greens and browns.
func Nature() *Theme {
	return &Theme{
		Trace:     rgb(101, 115, 126),
		Debug:     rgb(34, 139, 34),
		Info:      rgb(46, 139, 87),
		Warn:      rgb(218, 165, 32),
		Error:     rgb(139, 69, 19),
		Fatal:     rgbBgFg(139, 69, 19, 255, 255, 255),
		LogID:     rgb(101, 115, 126),
		Timestamp: rgb(101, 115, 126),
		DataKey:   rgb(34, 139, 34),
		DataValue: rgb(0, 0, 0),
	}
}

// Pastel is a soft theme with gentle, muted colors.
func Pastel() *Theme {
	return &Theme{
		Trace:     rgb(200, 200, 200),
		Debug:     rgb(173, 216, 230),
		Info:      rgb(144, 238, 144),
		Warn:      rgb(255, 218, 185),
		Error:     rgb(255, 182, 193),
		Fatal:     rgbBgFg(255, 182, 193, 105, 105, 105),
		LogID:     rgb(200, 200, 200),
		Timestamp: rgb(200, 200, 200),
		DataKey:   rgb(173, 216, 230),
		DataValue: rgb(105, 105, 105),
	}
}

// noColorTheme returns a theme that emits no escape codes — used when the
// transport is configured with NoColor: true.
func noColorTheme() *Theme {
	return &Theme{
		Trace:     noStyle,
		Debug:     noStyle,
		Info:      noStyle,
		Warn:      noStyle,
		Error:     noStyle,
		Fatal:     noStyle,
		LogID:     noStyle,
		Timestamp: noStyle,
		DataKey:   noStyle,
		DataValue: noStyle,
	}
}
