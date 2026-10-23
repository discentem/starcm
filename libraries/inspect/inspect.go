// Package inspect reports what a .star file loads and calls without running
// it, so tools can check configuration (e.g. "this file may only use
// download") before it's signed or deployed.
package inspect

import (
	"encoding/json"
	"sort"

	"go.starlark.net/syntax"
)

// Report describes one file.
type Report struct {
	File string `json:"file"`
	// Loads are the file's load() statements.
	Loads []Load `json:"loads"`
	// Calls are direct calls to names bound by load(), resolved through
	// aliases: load("starcm", run = "exec"); run(...) is a call to exec.
	Calls []Call `json:"calls"`
	// Values are uses of a loaded name other than a direct call: a dict like
	// vars["base"], or a function passed around, e.g. f = load_dynamic, whose
	// eventual calls can't be known statically.
	Values []Ref `json:"values"`
}

// Load is one load() statement.
type Load struct {
	Module string `json:"module"`
	Line   int32  `json:"line"`
	// Names maps each local name to the name it was loaded as.
	Names map[string]string `json:"names"`
}

// Ref is a loaded name used at a line.
type Ref struct {
	Module   string `json:"module"`
	Function string `json:"function"`
	Line     int32  `json:"line"`
}

// Call is a direct call to a loaded name.
type Call struct {
	Ref
	// Path is load_dynamic's module argument when it's a string literal, or
	// nil when it's computed at runtime. Only set for starcm's load_dynamic.
	Path *string
	// URL is the url argument of load_remote and install_remote_mac_pkg
	// when it's a string literal, or nil when it's computed at runtime. Only
	// set for those two starcm functions.
	URL *string
	// Inputs are the arguments whose values come from loaded names, such as
	// exec(args = ["-c", "echo " + vars["name"]]). Omitted when there are none.
	Inputs []Input
}

// MarshalJSON writes "path" for load_dynamic calls only, as null when the
// path is computed, so tools can tell "computed" from "not load_dynamic";
// "url" likewise for load_remote and install_remote_mac_pkg; and
// "inputs" only when there are some.
func (c Call) MarshalJSON() ([]byte, error) {
	out := struct {
		Ref
		Path   json.RawMessage `json:"path,omitempty"`
		URL    json.RawMessage `json:"url,omitempty"`
		Inputs []Input         `json:"inputs,omitempty"`
	}{Ref: c.Ref, Inputs: c.Inputs}
	if c.isLoadDynamic() {
		p, err := json.Marshal(c.Path) // nil marshals as null
		if err != nil {
			return nil, err
		}
		out.Path = p
	}
	if c.takesURL() {
		u, err := json.Marshal(c.URL) // nil marshals as null
		if err != nil {
			return nil, err
		}
		out.URL = u
	}
	return json.Marshal(out)
}

func (c Call) isLoadDynamic() bool { return c.Module == "starcm" && c.Function == "load_dynamic" }

// takesURL reports whether c fetches from a url argument.
func (c Call) takesURL() bool {
	return c.Module == "starcm" && (c.Function == "load_remote" || c.Function == "install_remote_mac_pkg")
}

// File parses src as starcm does and reports on it.
func File(filename string, src []byte) (*Report, error) {
	opts := &syntax.FileOptions{TopLevelControl: true} // as starcm's loader parses files
	f, err := opts.Parse(filename, src, 0)
	if err != nil {
		return nil, err
	}

	r := &Report{File: filename, Loads: []Load{}, Calls: []Call{}, Values: []Ref{}}
	type binding struct{ module, name string }
	bound := map[string]binding{}
	bindingIdents := map[*syntax.Ident]bool{}

	for _, stmt := range f.Stmts {
		load, ok := stmt.(*syntax.LoadStmt)
		if !ok {
			continue
		}
		l := Load{Module: load.ModuleName(), Line: line(load), Names: map[string]string{}}
		for i, from := range load.From {
			l.Names[load.To[i].Name] = from.Name
			bound[load.To[i].Name] = binding{l.Module, from.Name}
			bindingIdents[load.To[i]] = true
		}
		r.Loads = append(r.Loads, l)
	}

	loaded := map[string]Use{}
	for name, b := range bound {
		loaded[name] = Use{Module: b.module, Function: b.name}
	}
	fl := newFlows(f, loaded)

	called := map[*syntax.Ident]bool{}
	keywords := map[*syntax.Ident]bool{} // the "x" in f(x = 1) names a parameter, not a value
	syntax.Walk(f, func(n syntax.Node) bool {
		call, ok := n.(*syntax.CallExpr)
		if !ok {
			return true
		}
		for _, a := range call.Args {
			if kw, ok := a.(*syntax.BinaryExpr); ok && kw.Op == syntax.EQ {
				if id, ok := kw.X.(*syntax.Ident); ok {
					keywords[id] = true
				}
			}
		}
		fn, ok := call.Fn.(*syntax.Ident)
		if !ok {
			return true
		}
		b, ok := bound[fn.Name]
		if !ok {
			return true
		}
		called[fn] = true
		c := Call{Ref: Ref{Module: b.module, Function: b.name, Line: line(call)}, Inputs: fl.inputs(call)}
		if c.isLoadDynamic() {
			c.Path = literalString(firstArg(call, "module"))
		}
		if c.takesURL() {
			c.URL = literalString(firstArg(call, "url"))
		}
		r.Calls = append(r.Calls, c)
		return true
	})

	syntax.Walk(f, func(n syntax.Node) bool {
		id, ok := n.(*syntax.Ident)
		if !ok || bindingIdents[id] || called[id] || keywords[id] {
			return true
		}
		if b, ok := bound[id.Name]; ok {
			r.Values = append(r.Values, Ref{Module: b.module, Function: b.name, Line: line(id)})
		}
		return true
	})

	sort.SliceStable(r.Calls, func(i, j int) bool { return r.Calls[i].Line < r.Calls[j].Line })
	sort.SliceStable(r.Values, func(i, j int) bool { return r.Values[i].Line < r.Values[j].Line })
	return r, nil
}

// literalString returns e's value when it's a string literal, else nil.
func literalString(e syntax.Expr) *string {
	if lit, ok := e.(*syntax.Literal); ok && lit.Token == syntax.STRING {
		s := lit.Value.(string)
		return &s
	}
	return nil
}

// firstArg returns a call's argument named keyword, or else its first
// positional argument: load_dynamic's module, load_remote's url.
func firstArg(call *syntax.CallExpr, keyword string) syntax.Expr {
	var positional syntax.Expr
	for _, a := range call.Args {
		if kw, ok := a.(*syntax.BinaryExpr); ok && kw.Op == syntax.EQ {
			if id, ok := kw.X.(*syntax.Ident); ok && id.Name == keyword {
				return kw.Y
			}
			continue
		}
		if positional == nil {
			positional = a
		}
	}
	return positional
}

func line(n syntax.Node) int32 {
	start, _ := n.Span()
	return start.Line
}
