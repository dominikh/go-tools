// Package pkg ...
package pkg

func fn(x int) {
	switch x {
	}
	switch x {
	case 1:
	}

	switch x {
	case 1:
	case 2:
	case 3:
	}

	switch x {
	default:
	}

	switch x {
	default:
	case 1:
	}

	switch x {
	case 1:
	default:
	}

	switch x {
	case 1:
	default: //@ diag(`default case should be first or last in switch statement`)
	case 2:
	}

	// Don't flag either of these two; fallthrough is sensitive to the order of cases
	switch x {
	case 1:
		fallthrough
	default:
	case 2:
	}
	switch x {
	case 1:
	default:
		fallthrough
	case 2:
	}

	// Do flag these branches; don't get confused by the nested switch statements
	switch x {
	case 1:
		func() {
			switch x {
			case 1:
				fallthrough
			case 2:
			}
		}()
	default: //@ diag(`default case should be first or last in switch statement`)
	case 3:
	}

	switch x {
	case 1:
		switch x {
		case 1:
			fallthrough
		case 2:
		}
	default: //@ diag(`default case should be first or last in switch statement`)
	case 3:
	}

	// Fallthrough may be followed by empty statements
	switch x {
	case 1:
		fallthrough
		;
	default:
	case 3:
	}
}
