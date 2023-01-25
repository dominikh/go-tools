package quickfix

import (
	"testing"

	"honnef.co/go/tools/analysis/lint/testutil"
)

func TestAll(t *testing.T) {
	checks := map[string][]testutil.Test{
		"QF1001": {{Dir: "example.com/CheckDeMorgan"}},
		"QF1002": {{Dir: "example.com/CheckTaglessSwitch"}},
		"QF1003": {{Dir: "example.com/CheckIfElseToSwitch"}},
		"QF1004": {{Dir: "example.com/CheckStringsReplaceAll"}},
		"QF1005": {{Dir: "example.com/CheckMathPow"}},
		"QF1006": {{Dir: "example.com/CheckForLoopIfBreak"}},
		"QF1007": {{Dir: "example.com/CheckConditionalAssignment"}},
		"QF1008": {{Dir: "example.com/CheckExplicitEmbeddedSelector"}},
		"QF1009": {{Dir: "example.com/CheckTimeEquality"}},
		"QF1010": {{Dir: "example.com/CheckByteSlicePrinting"}},
		"QF1011": {{Dir: "example.com/CheckRedundantTypeInDeclaration"}},
		"QF1012": {{Dir: "example.com/CheckWriteBytesSprintf"}},
	}

	testutil.Run(t, Analyzers, checks)
}
