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
	"go.starlark.net/starlarkstruct"
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
	for i, v := range kwargs {
		if v[0].String() == fmt.Sprintf("\"%s\"", value) {
			return i, nil
		}
	}
	return -1, ErrIndexNotFound // Return -1 if the value is not found
}

func FindValueFromIndexInKwargs(kwargs []starlark.Tuple, index int) (*string, error) {
	if index == IndexNotFound {
		return nil, ErrIndexNotFound
	}
	s := kwargs[index][1].String()
	unquoted, _, _, err := Unquote(s)
	if err != nil {
		return nil, err
	}
	logging.Log("starlarkhelpers FindValueFromIndexInKwargs", deck.V(4), "unquoted", unquoted)
	return &unquoted, nil
}

func FindValueinKwargs(kwargs []starlark.Tuple, value string) (*string, error) {
	idx, err := FindIndexOfValueInKwargs(kwargs, value)
	if err != nil {
		if errors.Is(err, ErrIndexNotFound) {
			return nil, fmt.Errorf("%q not found in kwargs", value)
		}
		return nil, err
	}
	return FindValueFromIndexInKwargs(kwargs, idx)
}

func FindRawValueInKwargs(kwargs []starlark.Tuple, value string) (starlark.Value, error) {
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
	v, err := FindRawValueInKwargs(kwargs, value)
	if err != nil || v == nil {
		return nil, fmt.Errorf("error finding value %s in kwargs: %w", value, err)
	}
	// unquote the string
	s, _, _, err := Unquote(v.String())
	if err != nil {
		return nil, fmt.Errorf("error unquoting value %s: %w", v.String(), err)
	}
	return &s, nil
}

func FindIntInKwargs(kwargs []starlark.Tuple, value string, defaultValue int64) (int64, error) {
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

// ValueToGo converts a Starlark value to plain Go values (string, int64,
// float64, bool, nil, []any, map[string]any), recursing into containers.
// Structs, including starcm results, become maps of their fields.
func ValueToGo(value starlark.Value) (any, error) {
	switch value := value.(type) {
	case starlark.NoneType:
		return nil, nil
	case starlark.String:
		return string(value), nil
	case starlark.Int:
		i, ok := value.Int64()
		if !ok {
			return nil, fmt.Errorf("int %s is too large", value)
		}
		return i, nil
	case starlark.Float:
		return float64(value), nil
	case starlark.Bool:
		return bool(value), nil
	case *starlark.List, starlark.Tuple, *starlark.Set:
		return iterableToGo(value.(starlark.Iterable))
	case *starlark.Dict:
		return DictToGoMap(value)
	case *starlarkstruct.Struct:
		m := make(map[string]any)
		for _, name := range value.AttrNames() {
			attr, err := value.Attr(name)
			if err != nil {
				return nil, err
			}
			v, err := ValueToGo(attr)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", name, err)
			}
			m[name] = v
		}
		return m, nil
	default:
		return nil, fmt.Errorf("unsupported type %s", value.Type())
	}
}

func iterableToGo(value starlark.Iterable) ([]any, error) {
	list := []any{}
	iter := value.Iterate()
	defer iter.Done()
	var item starlark.Value
	for iter.Next(&item) {
		v, err := ValueToGo(item)
		if err != nil {
			return nil, fmt.Errorf("[%d]: %w", len(list), err)
		}
		list = append(list, v)
	}
	return list, nil
}

// DictToGoMap converts a Starlark dict with string keys using ValueToGo.
func DictToGoMap(dict *starlark.Dict) (map[string]any, error) {
	result := make(map[string]any)
	for _, item := range dict.Items() {
		key, ok := item[0].(starlark.String)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings, got %s key %s", item[0].Type(), item[0])
		}
		value, err := ValueToGo(item[1])
		if err != nil {
			return nil, fmt.Errorf("%q: %w", string(key), err)
		}
		result[string(key)] = value
	}
	return result, nil
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
