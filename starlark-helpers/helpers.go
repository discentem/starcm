package starlarkhelpers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/discentem/starcm/libraries/logging"
	"github.com/google/deck"
	"go.starlark.net/starlark"
)

type Function func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)

type LoaderFunc func(_ *starlark.Thread, module string) (starlark.StringDict, error)

const (
	IndexNotFound int = -1
)

var (
	ErrIndexNotFound = errors.New("index not found")
)

func FindIndexOfValueInKwargs(kwargs []starlark.Tuple, value string) (int, error) {
	logging.Log("starlarkhelpers FindIndexOfValueInKwargs", deck.V(3), "info", "value: %s", value)
	for i, v := range kwargs {
		if v[0].String() == fmt.Sprintf("\"%s\"", value) {
			return i, nil
		}
	}
	return -1, ErrIndexNotFound // Return -1 if the value is not found
}

func FindValueFromIndexInKwargs(kwargs []starlark.Tuple, index int) (*string, error) {
	logging.Log("starlarkhelpers FindValueFromIndexInKwargs", deck.V(3), "info", "index: %d", index)
	if index == IndexNotFound {
		return nil, ErrIndexNotFound
	}
	s := kwargs[index][1].String()
	logging.Log("starlarkhelpers FindValueFromIndexInKwargs", deck.V(3), "info", "index: %s", s)
	unquoted, _, _, err := Unquote(s)
	if err != nil {
		return nil, err
	}
	logging.Log("starlarkhelpers FindValueFromIndexInKwargs", deck.V(4), "unquoted", unquoted)
	return &unquoted, nil
}

func FindValueinKwargs(kwargs []starlark.Tuple, value string) (*string, error) {
	logging.Log("starlarkhelpers FindValueinKwargs", deck.V(3), "info", "value: %s", value)
	idx, err := FindIndexOfValueInKwargs(kwargs, value)
	if err != nil {
		return nil, err
	}
	return FindValueFromIndexInKwargs(kwargs, idx)
}

func FindRawValueInKwargs(kwargs []starlark.Tuple, value string) (starlark.Value, error) {
	logging.Log("starlarkhelpers FindRawValueInKwargs", deck.V(3), "info", "arg: %s", value)
	idx, err := FindIndexOfValueInKwargs(kwargs, value)
	if err != nil {
		return nil, err
	}
	if idx == IndexNotFound {
		return nil, err
	}
	return kwargs[idx][1], nil
}

func FindBoolInKwargs(kwargs []starlark.Tuple, value string, defaultValue bool) (bool, error) {
	logging.Log("starlarkhelpers FindBoolInKwargs", deck.V(3), "info", "value: %s", value)
	v, err := FindRawValueInKwargs(kwargs, value)
	if err != nil {
		if errors.Is(err, ErrIndexNotFound) {
			return defaultValue, nil
		}
		return defaultValue, err
	}
	if v == nil {
		return defaultValue, nil
	}
	return bool(v.Truth()), nil
}

func FindStringInKwargs(kwargs []starlark.Tuple, value string) (*string, error) {
	logging.Log("starlarkhelpers FindStringInKwargs", deck.V(3), "info", "value: %s", value)
	v, err := FindRawValueInKwargs(kwargs, value)
	if err != nil || v == nil {
		return nil, fmt.Errorf("error finding value %s in kwargs: %w", value, err)
	}
	// unquote the string
	s, _, _, err := Unquote(v.String())
	if err != nil {
		return nil, fmt.Errorf("error unquoting value %s: %w", v.String(), err)
	}
	logging.Log("starlarkhelpers FindStringInKwargs", deck.V(4), "unquoted", s)
	return &s, nil
}

func FindIntInKwargs(kwargs []starlark.Tuple, value string, defaultValue int64) (int64, error) {
	logging.Log("starlarkhelpers FindIntInKwargs", deck.V(3), "info", "value: %s", value)
	v, err := FindRawValueInKwargs(kwargs, value)
	if err != nil {
		if errors.Is(err, ErrIndexNotFound) {
			return defaultValue, nil
		}
		return defaultValue, err
	}
	if v == nil {
		return defaultValue, nil
	}
	var i int64
	if err := starlark.AsInt(v, &i); err != nil {
		return defaultValue, fmt.Errorf("error converting %s to int: %w", v.String(), err)
	}
	return i, nil
}

func FindValueInKwargsWithDefault(kwargs []starlark.Tuple, value string, defaultValue string) (*string, error) {
	idx, err := FindIndexOfValueInKwargs(kwargs, value)
	if err != nil && err != ErrIndexNotFound {
		return nil, err
	}
	if idx == IndexNotFound || err == ErrIndexNotFound {
		return &defaultValue, nil
	}
	return FindValueFromIndexInKwargs(kwargs, idx)
}

// Copied from https://github.com/google/starlark-go/blob/f457c4c2b267186711d0fadc15024e46b98186c5/syntax/quote.go

// unesc maps single-letter chars following \ to their actual values.
var unesc = [256]byte{
	'a':  '\a',
	'b':  '\b',
	'f':  '\f',
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
	'v':  '\v',
	'\\': '\\',
	'\'': '\'',
	'"':  '"',
}

func ValueToGo(value starlark.Value) any {
	switch value := value.(type) {
	case starlark.String:
		return string(value)
	case starlark.Int:
		i, _ := value.Int64()
		return i
	case starlark.Float:
		return float64(value)
	case starlark.Bool:
		return bool(value)
	case *starlark.List:
		list := make([]interface{}, value.Len())
		for i := 0; i < value.Len(); i++ {
			list[i] = ValueToGo(value.Index(i))
		}
		return list
	case *starlark.Dict:
		return DictToGoMap(value)
	default:
		return value
	}
}

func DictToGoMap(dict *starlark.Dict) map[string]interface{} {
	result := make(map[string]interface{})
	for _, item := range dict.Items() {
		key := item[0].(starlark.String)
		value := ValueToGo(item[1])
		result[string(key)] = value
	}
	return result
}

func OptionalKeyword(kw starlark.String) starlark.String {
	s := kw.GoString()
	if strings.HasSuffix(s, "?") {
		s := fmt.Sprintf("%s?", s)
		return starlark.String(s)
	}
	if strings.HasSuffix(s, "??") {
		return starlark.String(s)
	}
	return starlark.String(fmt.Sprintf("%s??", s))

}

// unquote unquotes the quoted string, returning the actual
// string value, whether the original was triple-quoted,
// whether it was a byte string, and an error describing invalid input.
func Unquote(quoted string) (s string, triple, isByte bool, err error) {
	// Check for raw prefix: means don't interpret the inner \.
	raw := false
	if strings.HasPrefix(quoted, "r") {
		raw = true
		quoted = quoted[1:]
	}
	// Check for bytes prefix.
	if strings.HasPrefix(quoted, "b") {
		isByte = true
		quoted = quoted[1:]
	}

	if len(quoted) < 2 {
		err = fmt.Errorf("string literal too short")
		return
	}

	if quoted[0] != '"' && quoted[0] != '\'' || quoted[0] != quoted[len(quoted)-1] {
		err = fmt.Errorf("string literal has invalid quotes")
		return
	}

	// Check for triple quoted string.
	quote := quoted[0]
	if len(quoted) >= 6 && quoted[1] == quote && quoted[2] == quote && quoted[:3] == quoted[len(quoted)-3:] {
		triple = true
		quoted = quoted[3 : len(quoted)-3]
	} else {
		quoted = quoted[1 : len(quoted)-1]
	}

	// Now quoted is the quoted data, but no quotes.
	// If we're in raw mode or there are no escapes or
	// carriage returns, we're done.
	var unquoteChars string
	if raw {
		unquoteChars = "\r"
	} else {
		unquoteChars = "\\\r"
	}
	if !strings.ContainsAny(quoted, unquoteChars) {
		s = quoted
		return
	}

	// Otherwise process quoted string.
	// Each iteration processes one escape sequence along with the
	// plain text leading up to it.
	buf := new(strings.Builder)
	for {
		// Remove prefix before escape sequence.
		i := strings.IndexAny(quoted, unquoteChars)
		if i < 0 {
			i = len(quoted)
		}
		buf.WriteString(quoted[:i])
		quoted = quoted[i:]

		if len(quoted) == 0 {
			break
		}

		// Process carriage return.
		if quoted[0] == '\r' {
			buf.WriteByte('\n')
			if len(quoted) > 1 && quoted[1] == '\n' {
				quoted = quoted[2:]
			} else {
				quoted = quoted[1:]
			}
			continue
		}

		// Process escape sequence.
		if len(quoted) == 1 {
			err = fmt.Errorf(`truncated escape sequence \`)
			return
		}

		switch quoted[1] {
		default:
			// In Starlark, like Go, a backslash must escape something.
			// (Python still treats unnecessary backslashes literally,
			// but since 3.6 has emitted a deprecation warning.)
			err = fmt.Errorf("invalid escape sequence \\%c", quoted[1])
			return

		case '\n':
			// Ignore the escape and the line break.
			quoted = quoted[2:]

		case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '\'', '"':
			// One-char escape.
			// Escapes are allowed for both kinds of quotation
			// mark, not just the kind in use.
			buf.WriteByte(unesc[quoted[1]])
			quoted = quoted[2:]

		case '0', '1', '2', '3', '4', '5', '6', '7':
			// Octal escape, up to 3 digits, \OOO.
			n := int(quoted[1] - '0')
			quoted = quoted[2:]
			for i := 1; i < 3; i++ {
				if len(quoted) == 0 || quoted[0] < '0' || '7' < quoted[0] {
					break
				}
				n = n*8 + int(quoted[0]-'0')
				quoted = quoted[1:]
			}
			if !isByte && n > 127 {
				err = fmt.Errorf(`non-ASCII octal escape \%o (use \u%04X for the UTF-8 encoding of U+%04X)`, n, n, n)
				return
			}
			if n >= 256 {
				// NOTE: Python silently discards the high bit,
				// so that '\541' == '\141' == 'a'.
				// Let's see if we can avoid doing that in BUILD files.
				err = fmt.Errorf(`invalid escape sequence \%03o`, n)
				return
			}
			buf.WriteByte(byte(n))

		case 'x':
			// Hexadecimal escape, exactly 2 digits, \xXX. [0-127]
			if len(quoted) < 4 {
				err = fmt.Errorf(`truncated escape sequence %s`, quoted)
				return
			}
			n, err1 := strconv.ParseUint(quoted[2:4], 16, 0)
			if err1 != nil {
				err = fmt.Errorf(`invalid escape sequence %s`, quoted[:4])
				return
			}
			if !isByte && n > 127 {
				err = fmt.Errorf(`non-ASCII hex escape %s (use \u%04X for the UTF-8 encoding of U+%04X)`,
					quoted[:4], n, n)
				return
			}
			buf.WriteByte(byte(n))
			quoted = quoted[4:]

		case 'u', 'U':
			// Unicode code point, 4 (\uXXXX) or 8 (\UXXXXXXXX) hex digits.
			sz := 6
			if quoted[1] == 'U' {
				sz = 10
			}
			if len(quoted) < sz {
				err = fmt.Errorf(`truncated escape sequence %s`, quoted)
				return
			}
			n, err1 := strconv.ParseUint(quoted[2:sz], 16, 0)
			if err1 != nil {
				err = fmt.Errorf(`invalid escape sequence %s`, quoted[:sz])
				return
			}
			if n > unicode.MaxRune {
				err = fmt.Errorf(`code point out of range: %s (max \U%08x)`,
					quoted[:sz], n)
				return
			}
			// As in Go, surrogates are disallowed.
			if 0xD800 <= n && n < 0xE000 {
				err = fmt.Errorf(`invalid Unicode code point U+%04X`, n)
				return
			}
			buf.WriteRune(rune(n))
			quoted = quoted[sz:]
		}
	}

	s = buf.String()
	return
}
