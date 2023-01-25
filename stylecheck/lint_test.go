package stylecheck

import (
	"testing"

	"honnef.co/go/tools/analysis/lint/testutil"
)

func TestAll(t *testing.T) {
	checks := map[string][]testutil.Test{
		"ST1000": {{Dir: "example.com/CheckPackageComment-1"}, {Dir: "example.com/CheckPackageComment-2"}, {Dir: "example.com/CheckPackageComment-3"}},
		"ST1001": {{Dir: "example.com/CheckDotImports"}},
		"ST1003": {{Dir: "example.com/CheckNames"}, {Dir: "example.com/CheckNames_generated"}},
		"ST1005": {{Dir: "example.com/CheckErrorStrings"}},
		"ST1006": {{Dir: "example.com/CheckReceiverNames"}},
		"ST1008": {{Dir: "example.com/CheckErrorReturn"}},
		"ST1011": {{Dir: "example.com/CheckTimeNames"}},
		"ST1012": {{Dir: "example.com/CheckErrorVarNames"}},
		"ST1013": {{Dir: "example.com/CheckHTTPStatusCodes"}},
		"ST1015": {{Dir: "example.com/CheckDefaultCaseOrder"}},
		"ST1016": {{Dir: "example.com/CheckReceiverNamesIdentical"}},
		"ST1017": {{Dir: "example.com/CheckYodaConditions"}},
		"ST1018": {{Dir: "example.com/CheckInvisibleCharacters"}},
		"ST1019": {{Dir: "example.com/CheckDuplicatedImports"}},
		"ST1020": {{Dir: "example.com/CheckExportedFunctionDocs"}},
		"ST1021": {{Dir: "example.com/CheckExportedTypeDocs"}},
		"ST1022": {{Dir: "example.com/CheckExportedVarDocs"}},
		"ST1023": {{Dir: "example.com/CheckRedundantTypeInDeclaration"}, {Dir: "example.com/CheckRedundantTypeInDeclaration_syscall"}},
	}

	testutil.Run(t, Analyzers, checks)
}
