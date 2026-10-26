package inspect

import (
	"sort"
	"strconv"

	"go.starlark.net/syntax"
)

// Input is one argument of a call and the loaded names its value comes from.
type Input struct {
	// Arg is the keyword, the position ("0", "1", ...) of a positional
	// argument, or "*" / "**" for unpacked arguments.
	Arg string `json:"arg"`
	// Uses are the loaded names the argument's value is computed from,
	// sorted: vars for vars["cmd"], read_file for read_file(...).
	Uses []Use `json:"uses"`
}

// Use is a loaded name a value comes from.
type Use struct {
	Module   string `json:"module"`
	Function string `json:"function"`
}

// flows finds which loaded names each local name's value can come from. It
// follows assignments (including +=, d[k] = v and x.append(v)), for loops,
// comprehensions, and the parameters and return values of the file's own
// functions. It goes by name and ignores order and scope, so it can report a
// use that can't happen but shouldn't miss one that can.
type flows struct {
	loaded map[string]Use
	// names maps a local name to the loaded names and local names its value
	// may come from.
	names map[string]*deps
	defs  map[string]*syntax.DefStmt
}

type deps struct {
	uses  map[Use]bool
	names map[string]bool
}

func newFlows(f *syntax.File, loaded map[string]Use) *flows {
	fl := &flows{loaded: loaded, names: map[string]*deps{}, defs: map[string]*syntax.DefStmt{}}
	syntax.Walk(f, func(n syntax.Node) bool {
		if def, ok := n.(*syntax.DefStmt); ok {
			fl.defs[def.Name.Name] = def
		}
		return true
	})
	syntax.Walk(f, func(n syntax.Node) bool {
		switch n := n.(type) {
		case *syntax.AssignStmt:
			fl.assign(n.LHS, n.RHS)
		case *syntax.ForStmt:
			fl.assign(n.Vars, n.X)
		case *syntax.ForClause:
			fl.assign(n.Vars, n.X)
		case *syntax.DefStmt:
			for _, p := range n.Params {
				if b, ok := p.(*syntax.BinaryExpr); ok && b.Op == syntax.EQ {
					fl.assign(b.X, b.Y) // a default value
				}
			}
			syntax.Walk(n, func(m syntax.Node) bool {
				if ret, ok := m.(*syntax.ReturnStmt); ok && ret.Result != nil {
					fl.assign(n.Name, ret.Result)
				}
				return true
			})
		case *syntax.CallExpr:
			fl.call(n)
		}
		return true
	})
	return fl
}

// call records what a call passes on: to the parameters of one of the file's
// own functions, or into the list or dict a mutating method is called on.
func (fl *flows) call(c *syntax.CallExpr) {
	switch fn := c.Fn.(type) {
	case *syntax.Ident:
		def, ok := fl.defs[fn.Name]
		if !ok {
			return
		}
		for _, p := range def.Params {
			for _, a := range c.Args {
				fl.assign(p, argValue(a))
			}
		}
	case *syntax.DotExpr:
		switch fn.Name.Name {
		case "append", "extend", "insert", "update", "setdefault":
			for _, a := range c.Args {
				fl.assign(fn.X, argValue(a))
			}
		}
	}
}

// assign records that the names in target may get their value from value.
// For d[k] or x.f, that's d or x.
func (fl *flows) assign(target, value syntax.Expr) {
	for _, name := range targetNames(target) {
		d := fl.names[name]
		if d == nil {
			d = &deps{uses: map[Use]bool{}, names: map[string]bool{}}
			fl.names[name] = d
		}
		fl.collect(value, d)
	}
}

// collect adds the names that value reads to d.
func (fl *flows) collect(value syntax.Expr, d *deps) {
	skip := map[*syntax.Ident]bool{}
	syntax.Walk(value, func(n syntax.Node) bool {
		switch n := n.(type) {
		case *syntax.DotExpr:
			skip[n.Name] = true // the field name in x.f
		case *syntax.CallExpr:
			for _, a := range n.Args {
				if kw, ok := a.(*syntax.BinaryExpr); ok && kw.Op == syntax.EQ {
					if id, ok := kw.X.(*syntax.Ident); ok {
						skip[id] = true // the keyword in f(k = v)
					}
				}
			}
		case *syntax.Ident:
			if skip[n] {
				return true
			}
			if u, ok := fl.loaded[n.Name]; ok {
				d.uses[u] = true
			} else {
				d.names[n.Name] = true
			}
		}
		return true
	})
}

// uses returns the loaded names value may come from, directly or through
// local names, sorted.
func (fl *flows) uses(value syntax.Expr) []Use {
	d := &deps{uses: map[Use]bool{}, names: map[string]bool{}}
	fl.collect(value, d)
	seen := map[string]bool{}
	queue := make([]string, 0, len(d.names))
	for n := range d.names {
		queue = append(queue, n)
	}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		if seen[n] {
			continue
		}
		seen[n] = true
		nd := fl.names[n]
		if nd == nil {
			continue
		}
		for u := range nd.uses {
			d.uses[u] = true
		}
		for m := range nd.names {
			queue = append(queue, m)
		}
	}
	out := make([]Use, 0, len(d.uses))
	for u := range d.uses {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Module != out[j].Module {
			return out[i].Module < out[j].Module
		}
		return out[i].Function < out[j].Function
	})
	return out
}

// inputs returns the arguments of c whose values come from loaded names.
func (fl *flows) inputs(c *syntax.CallExpr) []Input {
	var out []Input
	position := 0
	for _, a := range c.Args {
		var arg string
		switch a := a.(type) {
		case *syntax.BinaryExpr:
			if id, ok := a.X.(*syntax.Ident); ok && a.Op == syntax.EQ {
				arg = id.Name
			}
		case *syntax.UnaryExpr:
			switch a.Op {
			case syntax.STAR:
				arg = "*"
			case syntax.STARSTAR:
				arg = "**"
			}
		}
		if arg == "" {
			arg = strconv.Itoa(position)
			position++
		}
		if uses := fl.uses(argValue(a)); len(uses) > 0 {
			out = append(out, Input{Arg: arg, Uses: uses})
		}
	}
	return out
}

// argValue is an argument's value: v for k = v, x for *x and **x.
func argValue(a syntax.Expr) syntax.Expr {
	switch a := a.(type) {
	case *syntax.BinaryExpr:
		if _, ok := a.X.(*syntax.Ident); ok && a.Op == syntax.EQ {
			return a.Y
		}
	case *syntax.UnaryExpr:
		if a.Op == syntax.STAR || a.Op == syntax.STARSTAR {
			return a.X
		}
	}
	return a
}

// targetNames returns the names an assignment to target changes: x for x,
// a and b for a, b, d for d[k] and x for x.f. A parameter (x, x = default,
// *x, **x) is a target too.
func targetNames(target syntax.Expr) []string {
	switch t := target.(type) {
	case *syntax.Ident:
		return []string{t.Name}
	case *syntax.TupleExpr:
		return listNames(t.List)
	case *syntax.ListExpr:
		return listNames(t.List)
	case *syntax.ParenExpr:
		return targetNames(t.X)
	case *syntax.IndexExpr:
		return targetNames(t.X)
	case *syntax.SliceExpr:
		return targetNames(t.X)
	case *syntax.DotExpr:
		return targetNames(t.X)
	case *syntax.BinaryExpr:
		if t.Op == syntax.EQ {
			return targetNames(t.X)
		}
	case *syntax.UnaryExpr:
		if t.X != nil {
			return targetNames(t.X)
		}
	}
	return nil
}

func listNames(list []syntax.Expr) []string {
	var out []string
	for _, e := range list {
		out = append(out, targetNames(e)...)
	}
	return out
}
