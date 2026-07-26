package cli

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type Options struct {
	BaseZone        string
	JSON            bool
	NoColor         bool
	ForceColor      bool
	NoEmoji         bool
	FZF             bool
	Screenshot      bool
	ScreenshotTheme string
	Help            bool
	AddedZones      []string
	Expression      string
}

var timeExpression = regexp.MustCompile(`^([0-9]{1,2})(?:(?:h|:)([0-9]{0,2}))(?:(?:h|:)([0-9]{1,2}))?$`)

func Parse(args []string) (Options, error) {
	var options Options
	var expression []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(expression) > 0 {
			expression = append(expression, arg)
			continue
		}
		switch {
		case arg == "--help":
			options.Help = true
		case strings.HasPrefix(arg, "+") && len(arg) > 1:
			options.AddedZones = append(options.AddedZones, strings.TrimPrefix(arg, "+"))
		case strings.HasPrefix(arg, "-") && arg != "-":
			for position := 1; position < len(arg); position++ {
				option := arg[position]
				switch option {
				case 'h':
					options.Help = true
				case 'j':
					options.JSON = true
				case 'n':
					options.NoColor = true
				case 'C':
					options.ForceColor = true
				case 'E':
					options.NoEmoji = true
				case 'f':
					options.FZF = true
				case 's':
					options.Screenshot = true
				case 't', 'S':
					value := arg[position+1:]
					if value == "" {
						if i+1 >= len(args) {
							return Options{}, fmt.Errorf("-%c requires a value", option)
						}
						i++
						value = args[i]
					}
					if option == 't' {
						options.BaseZone = value
					} else {
						options.ScreenshotTheme = value
					}
					position = len(arg)
				default:
					return Options{}, fmt.Errorf("unknown option -%c", option)
				}
			}
		default:
			expression = append(expression, arg)
		}
	}
	if options.JSON && options.Screenshot {
		return Options{}, errors.New("-s and -j cannot be combined")
	}
	if options.ScreenshotTheme != "" && options.ScreenshotTheme != "light" && options.ScreenshotTheme != "dark" {
		return Options{}, errors.New("-S must be light or dark")
	}
	if len(expression) > 0 {
		match := timeExpression.FindStringSubmatch(expression[0])
		if match == nil {
			return Options{}, fmt.Errorf("invalid date format %q: specify a time first, such as 10h00 tomorrow", expression[0])
		}
		hour := match[1]
		minute := match[2]
		if minute == "" {
			minute = "00"
		} else if len(minute) == 1 {
			minute = "0" + minute
		}
		if len(match) > 3 && match[3] != "" {
			expression[0] = hour + ":" + minute + ":" + match[3]
		} else {
			expression[0] = hour + ":" + minute
		}
		options.Expression = strings.Join(expression, " ")
	}
	return options, nil
}
