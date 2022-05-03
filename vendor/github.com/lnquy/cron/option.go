package cron

// SetLogger allows the expression descriptor to output log via logger.
func SetLogger(logger Logger) Option {
	return func(exprDesc *ExpressionDescriptor) {
		exprDesc.logger = logger
	}
}

// Verbose sets the expression descriptor to output string in verbose format or not.
//
// Example: cronExpression = "* * 5 * * * *"
//  - verbose = false: Every second, between 05:00 and 05:59
//  - verbose = true: Every second, every minute, between 05:00 and 05:59, every day
func Verbose(v bool) Option {
	return func(exprDesc *ExpressionDescriptor) {
		exprDesc.isVerbose = v
	}
}

// DayOfWeekStartsAtOne configures first day of the week is Monday (index 1) or Sunday (index 0, default).
func DayOfWeekStartsAtOne(v bool) Option {
	return func(exprDesc *ExpressionDescriptor) {
		exprDesc.isDOWStartsAtOne = v
	}
}

// Use24HourTimeFormat configures the expression descriptor to output time in 24-hour format (14:00) or
// 12-hour format (2PM, default).
func Use24HourTimeFormat(v bool) Option {
	return func(exprDesc *ExpressionDescriptor) {
		exprDesc.is24HourTimeFormat = v
	}
}

// SetLocales initializes the list of initial locales that the expression descriptor will output in.
// By default, the expression descriptor always initialize English (Locale_en).
func SetLocales(locales ...LocaleType) Option {
	return func(exprDesc *ExpressionDescriptor) {
		loaders, err := NewLocaleLoaders(locales...)
		if err != nil {
			exprDesc.log("failed to init locale loaders: %s", err)
			return
		}

		if exprDesc.locales == nil {
			exprDesc.locales = make(map[LocaleType]Locale)
		}
		for _, loader := range loaders {
			exprDesc.locales[loader.GetLocaleType()] = loader
		}
	}
}
