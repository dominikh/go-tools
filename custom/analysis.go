package custom

import (
	"fmt"
	"plugin"

	"golang.org/x/tools/go/analysis"
)

// Analyzers opens the plugin at pluginPath and returns the Analyzers it contains.
func Analyzers(pluginPath string) (result map[string]*analysis.Analyzer, _ error) {
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, err
	}
	analyzersSymbol, err := p.Lookup("Analyzers")
	if err != nil {
		return nil, err
	}
	analyzers, ok := analyzersSymbol.(*map[string]*analysis.Analyzer)
	if !ok {
		return nil, fmt.Errorf("Analyzers must be of type %T, not %T.", result, analyzers)
	}
	return *analyzers, nil
}
