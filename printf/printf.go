// Package printf implements a parser for fmt.Printf-style format
// strings.
//
// It parses verbs according to the following syntax:
//
//	Numeric -> '0'-'9'
//	Letter -> 'a'-'z' | 'A'-'Z'
//	Index -> '[' Numeric+ ']'
//	Star -> '*'
//	Star -> Index '*'
//
//	Precision -> Numeric+ | Star
//	Width -> Numeric+ | Star
//
//	WidthAndPrecision -> Width '.' Precision
//	WidthAndPrecision -> Width '.'
//	WidthAndPrecision -> Width
//	WidthAndPrecision -> '.' Precision
//	WidthAndPrecision -> '.'
//
//	Flag -> '+' | '-' | '#' | ' ' | '0'
//	Verb -> Letter | '%'
//
//	Input -> '%' [ Flag+ ] [ WidthAndPrecision ] [ Index ] Verb
package printf

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// ErrInvalid is returned for invalid format strings or verbs.
var ErrInvalid = errors.New("invalid format string")

type Verb struct {
	Letter rune
	Flags  string

	Width     Argument
	Precision Argument
	// Which value in the argument list the verb uses.
	// -1 denotes the next argument,
	// values > 0 denote explicit arguments.
	// The value 0 denotes that no argument is consumed. This is the case for %%.
	Value int

	Raw string
}

// Argument is an implicit or explicit width or precision.
type Argument interface {
	isArgument()
}

// The Default value, when no width or precision is provided.
type Default struct{}

// Zero is the implicit zero value.
// This value may only appear for precisions in format strings like %6.f
type Zero struct{}

// Star is a * value, which may either refer to the next argument (Index == -1) or an explicit argument.
type Star struct{ Index int }

// A Literal value, such as 6 in %6d.
type Literal int

func (Default) isArgument() {}
func (Zero) isArgument()    {}
func (Star) isArgument()    {}
func (Literal) isArgument() {}

// Parse parses f and returns a list of actions.
// An action may either be a literal string, or a Verb.
func Parse(formatString string) ([]interface{}, error) {
	var result []interface{}
	for len(formatString) > 0 {
		if formatString[0] == '%' {
			verb, consumed, err := ParseVerb(formatString)
			if err != nil {
				return nil, err
			}
			formatString = formatString[consumed:]
			result = append(result, verb)
		} else {
			nextPrecent := strings.IndexByte(formatString, '%')
			if nextPrecent > -1 {
				result = append(result, formatString[:nextPrecent])
				formatString = formatString[nextPrecent:]
			} else {
				result = append(result, formatString)
				formatString = ""
			}
		}
	}

	return result, nil
}

// ParseVerb parses the verb at the beginning of f.
// It returns the verb, how much of the input was consumed, and an error, if any.
func ParseVerb(formatString string) (Verb, int, error) {
	if len(formatString) < 2 {
		return Verb{}, 0, ErrInvalid
	}

	matches := re.FindStringSubmatch(formatString)
	if matches == nil {
		return Verb{}, 0, ErrInvalid
	}

	verb := Verb{
		Letter: []rune(matches[groupVerb])[0],
		Flags:  matches[groupFlags],
		Raw:    matches[0],
	}

	// Parse width
	verb.Width = parseWidth(matches[groupWidth], matches[groupWidthIndex], matches[groupWidthStar])

	// Parse precision
	verb.Precision = parsePrecision(matches[groupDot], matches[groupPrecision], matches[groupPrecisionIndex])

	// Determine value
	verb.Value = determineValue(matches[groupVerb], matches[groupVerbIndex])

	return verb, len(matches[0]), nil
}

// parseWidth parses the width component of a verb.
func parseWidth(literal, index, starIndex string) Argument {
	if literal != "" {
		// Literal width
		return Literal(atoi(literal))
	}
	if starIndex != "" {
		// Star width
		if index != "" {
			return Star{atoi(index)}
		} else {
			return Star{-1}
		}
	}
	// Default width
	return Default{}

}

// parsePrecision parses the precision component of a verb.
func parsePrecision(dot, literal, index string) Argument {
	if dot == "" {
		return Default{}
	}
	if literal != "" {
		return Literal(atoi(literal))
	}
	if index != "" {
		return Star{Index: atoi(index)}
	}
	return Zero{}
}

// determineValue determines the value index based on the verb and index match.
func determineValue(verb, index string) int {
	if verb == "%" {
		return 0
	}
	if index != "" {
		return atoi(index)
	}
	return -1
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// Regex group indices.
const (
	groupFlags          = 1
	groupWidth          = 2
	groupWidthStar      = 3
	groupWidthIndex     = 5
	groupDot            = 6
	groupPrecision      = 7
	groupPrecisionIndex = 10
	groupVerbIndex      = 11
	groupVerb           = 12
)

// Regular expressions for parsing.
const (
	regexFlags             = `([+#0 -]*)`
	regexVerb              = `([a-zA-Z%])`
	regexIndex             = `(?:\[([0-9]+)\])`
	regexStar              = `((` + regexIndex + `)?\*)`
	regexWidthLiteral      = `([0-9]+)`
	regexWidthStar         = regexStar
	regexWidth             = `(?:` + regexWidthLiteral + `|` + regexWidthStar + `)`
	regexPrecision         = regexWidth
	regexWidthAndPrecision = `(?:(?:` + regexWidth + `)?(?:(\.)(?:` + regexPrecision + `)?)?)`
)

var re = regexp.MustCompile(`^%` + regexFlags + regexWidthAndPrecision + `?` + regexIndex + `?` + regexVerb)
