package config

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
)

type config struct {
	cfg  Config
	meta toml.MetaData
}

func mergeLists(a, b []string) []string {
	out := make([]string, 0, len(a)+len(b))
	for _, el := range b {
		if el == "inherit" {
			out = append(out, a...)
		} else {
			out = append(out, el)
		}
	}
	return out
}

func normalizeList(list []string) []string {
	if len(list) > 1 {
		sort.Strings(list)
		nlist := make([]string, 0, len(list))
		nlist = append(nlist, list[0])
		for i, el := range list[1:] {
			if el != list[i] {
				nlist = append(nlist, el)
			}
		}
		list = nlist
	}

	for _, el := range list {
		if el == "inherit" {
			// This should never happen, because the default config
			// should not use "inherit"
			panic(`unresolved "inherit"`)
		}
		if el == "all" {
			return []string{"all"}
		}
	}

	return list
}

func (cfg config) Merge(ocfg config) config {
	if ocfg.meta.IsDefined("staticcheck", "enabled_checks") {
		cfg.cfg.Staticcheck.EnabledChecks = mergeLists(cfg.cfg.Staticcheck.EnabledChecks, ocfg.cfg.Staticcheck.EnabledChecks)
	}
	if ocfg.meta.IsDefined("staticcheck", "disabled_checks") {
		cfg.cfg.Staticcheck.DisabledChecks = mergeLists(cfg.cfg.Staticcheck.DisabledChecks, ocfg.cfg.Staticcheck.DisabledChecks)
	}

	if ocfg.meta.IsDefined("simple", "enabled_checks") {
		cfg.cfg.Simple.EnabledChecks = mergeLists(cfg.cfg.Simple.EnabledChecks, ocfg.cfg.Simple.EnabledChecks)
	}
	if ocfg.meta.IsDefined("simple", "disabled_checks") {
		cfg.cfg.Simple.DisabledChecks = mergeLists(cfg.cfg.Simple.DisabledChecks, ocfg.cfg.Simple.DisabledChecks)
	}

	if ocfg.meta.IsDefined("unused", "enabled_checks") {
		cfg.cfg.Unused.EnabledChecks = mergeLists(cfg.cfg.Unused.EnabledChecks, ocfg.cfg.Unused.EnabledChecks)
	}
	if ocfg.meta.IsDefined("unused", "disabled_checks") {
		cfg.cfg.Unused.DisabledChecks = mergeLists(cfg.cfg.Unused.DisabledChecks, ocfg.cfg.Unused.DisabledChecks)
	}

	if ocfg.meta.IsDefined("errcheck", "enabled_checks") {
		cfg.cfg.Errcheck.EnabledChecks = mergeLists(cfg.cfg.Errcheck.EnabledChecks, ocfg.cfg.Errcheck.EnabledChecks)
	}
	if ocfg.meta.IsDefined("errcheck", "disabled_checks") {
		cfg.cfg.Errcheck.DisabledChecks = mergeLists(cfg.cfg.Errcheck.DisabledChecks, ocfg.cfg.Errcheck.DisabledChecks)
	}

	if ocfg.meta.IsDefined("stylecheck", "enabled_checks") {
		cfg.cfg.Stylecheck.EnabledChecks = mergeLists(cfg.cfg.Stylecheck.EnabledChecks, ocfg.cfg.Stylecheck.EnabledChecks)
	}
	if ocfg.meta.IsDefined("stylecheck", "disabled_checks") {
		cfg.cfg.Stylecheck.DisabledChecks = mergeLists(cfg.cfg.Stylecheck.DisabledChecks, ocfg.cfg.Stylecheck.DisabledChecks)
	}
	if ocfg.meta.IsDefined("stylecheck", "initialisms") {
		cfg.cfg.Stylecheck.Initialisms = mergeLists(cfg.cfg.Stylecheck.Initialisms, ocfg.cfg.Stylecheck.Initialisms)
	}
	if ocfg.meta.IsDefined("stylecheck", "dot_import_whitelist") {
		cfg.cfg.Stylecheck.DotImportWhitelist = mergeLists(cfg.cfg.Stylecheck.DotImportWhitelist, ocfg.cfg.Stylecheck.DotImportWhitelist)
	}
	if ocfg.meta.IsDefined("stylecheck", "http_status_code_whitelist") {
		cfg.cfg.Stylecheck.HTTPStatusCodeWhitelist = mergeLists(cfg.cfg.Stylecheck.HTTPStatusCodeWhitelist, ocfg.cfg.Stylecheck.HTTPStatusCodeWhitelist)
	}
	return cfg
}

type Config struct {
	// TODO(dh): this implementation makes it impossible for external
	// clients to add their own checkers with configuration. At the
	// moment, we don't really care about that; we don't encourage
	// that people use this package. In the future, we may. The
	// obvious solution would be using map[string]interface{}, but
	// that's obviously subpar.
	General     GeneralConfig     `toml:"general"`
	Staticcheck StaticcheckConfig `toml:"staticcheck"`
	Simple      SimpleConfig      `toml:"simple"`
	Unused      UnusedConfig      `toml:"unused"`
	Errcheck    ErrcheckConfig    `toml:"errcheck"`
	Stylecheck  StylecheckConfig  `toml:"stylecheck"`
}

type GeneralConfig struct{}

type Checklist struct {
	EnabledChecks  []string `toml:"enabled_checks"`
	DisabledChecks []string `toml:"disabled_checks"`
}

type StaticcheckConfig struct {
	Checklist
}

type SimpleConfig struct {
	Checklist
}

type UnusedConfig struct {
	Checklist
}

type ErrcheckConfig struct {
	Checklist
}

type StylecheckConfig struct {
	Checklist
	Initialisms             []string `toml:"initialisms"`
	DotImportWhitelist      []string `toml:"dot_import_whitelist"`
	HTTPStatusCodeWhitelist []string `toml:"http_status_code_whitelist"`
}

var defaultConfig = Config{
	General:     defaultGeneralConfig,
	Staticcheck: defaultStaticcheckConfig,
	Simple:      defaultSimpleConfig,
	Unused:      defaultUnusedConfig,
	Errcheck:    defaultErrcheckConfig,
	Stylecheck:  defaultStylecheckConfig,
}

var defaultGeneralConfig = GeneralConfig{}

var defaultStaticcheckConfig = StaticcheckConfig{
	Checklist: Checklist{
		EnabledChecks:  []string{"all"},
		DisabledChecks: []string{},
	},
}

var defaultSimpleConfig = SimpleConfig{
	Checklist: Checklist{
		EnabledChecks:  []string{"all"},
		DisabledChecks: []string{},
	},
}

var defaultUnusedConfig = UnusedConfig{
	Checklist: Checklist{
		EnabledChecks:  []string{"all"},
		DisabledChecks: []string{},
	},
}

var defaultErrcheckConfig = ErrcheckConfig{
	Checklist: Checklist{
		EnabledChecks:  []string{"all"},
		DisabledChecks: []string{},
	},
}

var defaultStylecheckConfig = StylecheckConfig{
	Checklist: Checklist{
		EnabledChecks:  []string{"all"},
		DisabledChecks: []string{},
	},
	Initialisms: []string{
		"ACL", "API", "ASCII", "CPU", "CSS", "DNS",
		"EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID",
		"IP", "JSON", "QPS", "RAM", "RPC", "SLA",
		"SMTP", "SQL", "SSH", "TCP", "TLS", "TTL",
		"UDP", "UI", "GID", "UID", "UUID", "URI",
		"URL", "UTF8", "VM", "XML", "XMPP", "XSRF",
		"XSS",
	},
	DotImportWhitelist:      []string{},
	HTTPStatusCodeWhitelist: []string{"200", "400", "404", "500"},
}

const configName = "staticcheck.conf"

func parseConfigs(dir string) ([]config, error) {
	var out []config

	// TODO(dh): consider stopping at the GOPATH/module boundary
	for dir != "" {
		f, err := os.Open(filepath.Join(dir, configName))
		if os.IsNotExist(err) {
			ndir := filepath.Dir(dir)
			if ndir == dir {
				break
			}
			dir = ndir
			continue
		}
		if err != nil {
			return nil, err
		}
		var cfg Config
		meta, err := toml.DecodeReader(f, &cfg)
		f.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, config{cfg, meta})
		ndir := filepath.Dir(dir)
		if ndir == dir {
			break
		}
		dir = ndir
	}
	out = append(out, config{
		cfg:  defaultConfig,
		meta: toml.MetaData{}, // meta of the base config should never be accessed
	})
	if len(out) < 2 {
		return out, nil
	}
	for i := 0; i < len(out)/2; i++ {
		out[i], out[len(out)-1-i] = out[len(out)-1-i], out[i]
	}
	return out, nil
}

func mergeConfigs(confs []config) Config {
	if len(confs) == 0 {
		// This shouldn't happen because we always have at least a
		// default config.
		panic("trying to merge zero configs")
	}
	if len(confs) == 1 {
		return confs[0].cfg
	}
	conf := confs[0]
	for _, oconf := range confs[1:] {
		conf = conf.Merge(oconf)
	}
	return conf.cfg
}

func Load(dir string) (Config, error) {
	confs, err := parseConfigs(dir)
	if err != nil {
		return Config{}, err
	}
	conf := mergeConfigs(confs)

	conf.Staticcheck.EnabledChecks = normalizeList(conf.Staticcheck.EnabledChecks)
	conf.Staticcheck.DisabledChecks = normalizeList(conf.Staticcheck.DisabledChecks)

	conf.Simple.EnabledChecks = normalizeList(conf.Simple.EnabledChecks)
	conf.Simple.DisabledChecks = normalizeList(conf.Simple.DisabledChecks)

	conf.Unused.EnabledChecks = normalizeList(conf.Unused.EnabledChecks)
	conf.Unused.DisabledChecks = normalizeList(conf.Unused.DisabledChecks)

	conf.Errcheck.EnabledChecks = normalizeList(conf.Errcheck.EnabledChecks)
	conf.Errcheck.DisabledChecks = normalizeList(conf.Errcheck.DisabledChecks)

	conf.Stylecheck.EnabledChecks = normalizeList(conf.Stylecheck.EnabledChecks)
	conf.Stylecheck.DisabledChecks = normalizeList(conf.Stylecheck.DisabledChecks)
	conf.Stylecheck.Initialisms = normalizeList(conf.Stylecheck.Initialisms)
	conf.Stylecheck.DotImportWhitelist = normalizeList(conf.Stylecheck.DotImportWhitelist)

	return conf, nil
}
