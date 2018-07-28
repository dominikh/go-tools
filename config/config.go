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
	return cfg
}

type Config struct {
	Stylecheck StylecheckConfig
}

type StylecheckConfig struct {
	EnabledChecks      []string `toml:"enabled_checks"`
	DisabledChecks     []string `toml:"disabled_checks"`
	Initialisms        []string `toml:"initialisms"`
	DotImportWhitelist []string `toml:"dot_import_whitelist"`
}

var defaultConfig = Config{
	Stylecheck: defaultStylecheckConfig,
}

var defaultStylecheckConfig = StylecheckConfig{
	EnabledChecks:  []string{"all"},
	DisabledChecks: []string{},
	Initialisms: []string{
		"ACL", "API", "ASCII", "CPU", "CSS", "DNS",
		"EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID",
		"IP", "JSON", "QPS", "RAM", "RPC", "SLA",
		"SMTP", "SQL", "SSH", "TCP", "TLS", "TTL",
		"UDP", "UI", "GID", "UID", "UUID", "URI",
		"URL", "UTF8", "VM", "XML", "XMPP", "XSRF",
		"XSS",
	},
	DotImportWhitelist: []string{},
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
	conf.Stylecheck.EnabledChecks = normalizeList(conf.Stylecheck.EnabledChecks)
	conf.Stylecheck.DisabledChecks = normalizeList(conf.Stylecheck.DisabledChecks)
	conf.Stylecheck.Initialisms = normalizeList(conf.Stylecheck.Initialisms)
	conf.Stylecheck.DotImportWhitelist = normalizeList(conf.Stylecheck.DotImportWhitelist)

	return conf, nil
}
