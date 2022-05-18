package ir

// import (
// 	"fmt"
// 	"go/constant"
// 	"go/token"
// 	"log"
// )

// type Mapping[D any] struct {
// 	V Value
// 	D D
// }

// func (m Mapping[D]) String() string {
// 	return fmt.Sprintf("%s = %s", m.V.Name(), m.D)
// }

// type DFA[D any] struct {
// 	Join     func(D, D) D
// 	Greater  func(D, D) bool
// 	Transfer func(Instruction) []Mapping[D]
// 	Top      D
// 	Bottom   D

// 	Mapping map[Value]D
// }

// func (dfa *DFA[D]) Value(v Value) D {
// 	d, ok := dfa.Mapping[v]
// 	if ok {
// 		return d
// 	} else {
// 		return dfa.Bottom
// 	}
// }

// func dfs[D any](fn *Function, dfa *DFA[D]) {
// 	dfa.Mapping = map[Value]D{}

// 	var worklist []*BasicBlock
// 	worklist = append(worklist, fn.Blocks...)
// 	for len(worklist) > 0 {
// 		b := worklist[len(worklist)-1]
// 		worklist = worklist[:len(worklist)-1]

// 		for _, instr := range b.Instrs {
// 			ds := dfa.Transfer(instr)
// 			log.Printf("transfer(%s) = %s", instr, ds)
// 			for _, d := range ds {
// 				old := dfa.Value(d.V)
// 				dd := d.D
// 				greater := dfa.Greater(dd, old)
// 				log.Printf("greater(%s, %s) = %t", dd, old, greater)
// 				if greater {
// 					dfa.Mapping[d.V] = dd
// 					log.Println(d.V, *instr.Referrers())
// 					for _, ref := range *instr.Referrers() {
// 						worklist = append(worklist, ref.Block())
// 					}
// 				}
// 			}
// 		}
// 	}
// }

// type extremum struct{ _ *[]byte }

// var top = &extremum{}
// var bottom = &extremum{}

// func (ex *extremum) String() string {
// 	if ex == top {
// 		return "⊤"
// 	} else if ex == bottom {
// 		return "⊥"
// 	} else {
// 		panic("XXX")
// 	}
// }

// func xxx(fn *Function) {
// 	type constantD interface{} // constant.Value, top, bottom

// 	var dfa *DFA[constantD]
// 	dfa = &DFA[constantD]{
// 		Join: func(a, b constantD) constantD {
// 			if a == top || b == top {
// 				return top
// 			}
// 			if a == bottom {
// 				return b
// 			}
// 			if b == bottom {
// 				return a
// 			}
// 			if a == b {
// 				return a
// 			}
// 			return top
// 		},
// 		// a > b
// 		Greater: func(a, b constantD) bool {
// 			if a == b {
// 				return false
// 			}
// 			if b == top {
// 				return false
// 			}
// 			if a == bottom {
// 				return false
// 			}
// 			return true
// 		},
// 		Transfer: func(instr Instruction) []Mapping[constantD] {
// 			switch instr := instr.(type) {
// 			case *Phi:
// 				var d constantD = bottom
// 				for _, edge := range instr.Edges {
// 					d = dfa.Join(d, dfa.Value(edge))
// 				}
// 				return []Mapping[constantD]{{instr, d}}
// 			case *Sigma:
// 				// XXX consider branch conditions
// 				return []Mapping[constantD]{{instr, dfa.Value(instr.X)}}
// 			case *Const:
// 				return []Mapping[constantD]{{instr, instr.Value}}
// 			case *BinOp:
// 				if instr.Op == token.ADD {
// 					x := dfa.Value(instr.X)
// 					y := dfa.Value(instr.Y)

// 					if x == top || y == top {
// 						return []Mapping[constantD]{{instr, top}}
// 					}
// 					if x == bottom || y == bottom {
// 						return nil
// 					}
// 					return []Mapping[constantD]{{instr, constant.BinaryOp(x.(constant.Value), instr.Op, y.(constant.Value))}}
// 				} else {
// 					return nil
// 				}
// 			default:
// 				return nil
// 			}
// 		},
// 		Top:    top,
// 		Bottom: bottom,
// 	}
// 	dfs(fn, dfa)

// 	log.Println(dfa.Mapping)
// }
