package ir

func (b *builder) buildExits(fn *Function) {
	if fn.Package().Pkg.Path() == "runtime" {
		switch fn.Name() {
		case "exit":
			fn.WillExit = true
			return
		case "throw":
			fn.WillExit = true
			return
		case "Goexit":
			fn.WillUnwind = true
			return
		}
	}
	buildDomTree(fn)

	isRecoverCall := func(instr Instruction) bool {
		if instr, ok := instr.(*Call); ok {
			if builtin, ok := instr.Call.Value.(*Builtin); ok {
				if builtin.Name() == "recover" {
					return true
				}
			}
		}
		return false
	}

	// All panics branch to the exit block, which means that if every
	// possible path through the function panics, then all
	// predecessors of the exit block must panic.
	willPanic := true
	for _, pred := range fn.Exit.Preds {
		if _, ok := pred.Control().(*Panic); !ok {
			willPanic = false
		}
	}
	if willPanic {
		recovers := false
	recoverLoop:
		for _, u := range fn.Blocks {
			for _, instr := range u.Instrs {
				if instr, ok := instr.(*Defer); ok {
					call := instr.Call.StaticCallee()
					if call == nil {
						// not a static call, so we can't be sure the
						// deferred call isn't calling recover
						recovers = true
						break recoverLoop
					}
					if len(call.Blocks) == 0 {
						// external function, we don't know what's
						// happening inside it
						//
						// TODO(dh): this includes functions from
						// imported packages, due to how go/analysis
						// works. We could introduce another fact,
						// like we've done for exiting and unwinding,
						// but it doesn't seem worth it. Virtually all
						// uses of recover will be in closures.
						recovers = true
						break recoverLoop
					}
					for _, y := range call.Blocks {
						for _, instr2 := range y.Instrs {
							if isRecoverCall(instr2) {
								recovers = true
								break recoverLoop
							}
						}
					}
				}
			}
		}
		if !recovers {
			fn.WillUnwind = true
			return
		}
	}

	// TODO(dh): don't check that any specific call dominates the exit
	// block. instead, check that all calls combined cover every
	// possible path through the function.
	exits := NewBlockSet(len(fn.Blocks))
	unwinds := NewBlockSet(len(fn.Blocks))
	for _, u := range fn.Blocks {
		for _, instr := range u.Instrs {
			if instr, ok := instr.(CallInstruction); ok {
				switch instr.(type) {
				case *Defer, *Call:
				default:
					continue
				}
				if instr.Common().IsInvoke() {
					// give up
					return
				}
				var call *Function
				switch instr.Common().Value.(type) {
				case *Function, *MakeClosure:
					call = instr.Common().StaticCallee()
				case *Builtin:
					// the only builtins that affect control flow are
					// panic and recover, and we've already handled
					// those
					continue
				default:
					// dynamic dispatch
					return
				}
				// buildFunction is idempotent. if we're part of a
				// (mutually) recursive call chain, then buildFunction
				// will immediately return, and fn.WillExit will be false.
				if call.Package() == fn.Package() {
					b.buildFunction(call)
				}
				dom := u.Dominates(fn.Exit)
				if call.WillExit {
					if dom {
						fn.WillExit = true
						return
					}
					exits.Add(u)
				} else if call.WillUnwind {
					if dom {
						fn.WillUnwind = true
						return
					}
					unwinds.Add(u)
				}
			}
		}
	}

	// depth-first search trying to find a path to the exit block that
	// doesn't cross any of the blacklisted blocks
	seen := NewBlockSet(len(fn.Blocks))
	var findPath func(root *BasicBlock, bl *BlockSet) bool
	findPath = func(root *BasicBlock, bl *BlockSet) bool {
		if root == fn.Exit {
			return true
		}
		if seen.Has(root) {
			return false
		}
		if bl.Has(root) {
			return false
		}
		seen.Add(root)
		for _, succ := range root.Succs {
			if findPath(succ, bl) {
				return true
			}
		}
		return false
	}

	if exits.Num() > 0 {
		if !findPath(fn.Blocks[0], exits) {
			fn.WillExit = true
			return
		}
	}
	if unwinds.Num() > 0 {
		if !findPath(fn.Blocks[0], unwinds) {
			fn.WillUnwind = true
			return
		}
	}
}

func (b *builder) addUnreachables(fn *Function) {
	for _, bb := range fn.Blocks {
		for i, instr := range bb.Instrs {
			if instr, ok := instr.(*Call); ok {
				var call *Function
				switch v := instr.Common().Value.(type) {
				case *Function:
					call = v
				case *MakeClosure:
					call = v.Fn.(*Function)
				}
				if call == nil {
					continue
				}
				if call.Package() == fn.Package() {
					// make sure we have information on all functions in this package
					b.buildFunction(call)
				}
				if call.WillExit {
					// This call will cause the process to terminate.
					// Remove remaining instructions in the block and
					// replace any control flow with Unreachable.
					for _, succ := range bb.Succs {
						succ.removePred(bb)
					}
					bb.Succs = bb.Succs[:0]

					bb.Instrs = bb.Instrs[:i+1]
					bb.emit(new(Unreachable), instr.Source())
					addEdge(bb, fn.Exit)
					break
				} else if call.WillUnwind {
					// This call will cause the goroutine to terminate
					// and defers to run (i.e. a panic or
					// runtime.Goexit). Remove remaining instructions
					// in the block and replace any control flow with
					// an unconditional jump to the exit block.
					for _, succ := range bb.Succs {
						succ.removePred(bb)
					}
					bb.Succs = bb.Succs[:0]

					bb.Instrs = bb.Instrs[:i+1]
					bb.emit(new(Jump), instr.Source())
					addEdge(bb, fn.Exit)
					break
				}
			}
		}
	}
}
