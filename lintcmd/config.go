package lintcmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

func parseBuildConfigs(r io.Reader) ([]BuildConfig, error) {
	var builds []BuildConfig
	br := bufio.NewReader(r)
	i := 0
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, envs, flags, err := parseBuildConfig(line)
		if err != nil {
			if err, ok := err.(parseBuildConfigError); ok {
				err.line = i
				return nil, err
			} else {
				return nil, err
			}
		}

		bc := BuildConfig{
			Name:  name,
			Envs:  make([]string, 0, len(envs)),
			Flags: make([]string, 0, len(flags)),
		}
		for _, env := range envs {
			bc.Envs = append(bc.Envs, fmt.Sprintf("%s=%s", env[0], env[1]))
		}
		for _, flag := range flags {
			bc.Flags = append(bc.Flags, flag[0], flag[1])
		}
		builds = append(builds, bc)

		i++
	}
	return builds, nil
}

type parseBuildConfigError struct {
	line   int // 0-based line number
	offset int // 0-based offset
	msg    string
}

func (err parseBuildConfigError) Error() string { return err.msg }

func parseBuildConfig(line string) (name string, envs, flags [][2]string, err error) {
	if line == "" {
		return "", nil, nil, errors.New("couldn't parse empty build config")
	}
	if strings.Index(line, ":") == len(line)-1 {
		name = line[:len(line)-1]
	} else {
		idx := strings.Index(line, ": ")
		if idx == -1 {
			return name, envs, flags, parseBuildConfigError{0, 0, "missing build name"}
		}
		name = line[:idx]
		off := idx + 2
		line = strings.TrimSpace(line[idx+2:])

		const (
			stateStart = iota
			stateEnvName
			stateEnvValueStart
			stateEnvValueQuoted
			stateEnvValue
			stateFlagNameStart
			stateFlagName
			stateFlagValueStart
			stateFlagValueQuoted
			stateFlagValue
		)

		state := stateStart
		start := 0
		var valL string
		for i, r := range line {
			switch state {
			case stateStart:
				valL = ""

				if r == '-' {
					state = stateFlagNameStart
					start = i
				} else if r == ' ' {
				} else if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' {
					state = stateEnvName
					start = i
				} else {
					return name, envs, flags, parseBuildConfigError{0, i + off, fmt.Sprintf("expected start of environment variable or flag, got %q", r)}
				}
			case stateEnvName:
				if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' {
				} else if r == '=' {
					valL = line[start:i]
					state = stateEnvValueStart
					start = i + 1
				} else {
					return name, envs, flags, parseBuildConfigError{0, i + off, fmt.Sprintf("invalid character %q in environment variable name", r)}
				}
			case stateEnvValueStart:
				if r == '"' {
					state = stateEnvValueQuoted
					start = i + 1
				} else if r != ' ' {
					state = stateEnvValue
				} else {
					// empty value
					envs = append(envs, [2]string{valL, ""})
					state = stateStart
				}
			case stateEnvValueQuoted:
				if r == '"' {
					// end of value
					envs = append(envs, [2]string{valL, line[start:i]})
					state = stateStart
				}
			case stateEnvValue:
				if r == ' ' {
					// end of value
					envs = append(envs, [2]string{valL, line[start:i]})
					state = stateStart
				}
			case stateFlagNameStart:
				if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
					state = stateFlagName
				} else {
					return name, envs, flags, parseBuildConfigError{0, i + off, fmt.Sprintf("invalid character %q in flag name", r)}
				}
			case stateFlagName:
				if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
				} else if r == ' ' {
					// flag without value
					flags = append(flags, [2]string{line[start:i], ""})
					state = stateStart
				} else if r == '=' {
					// flag with value
					valL = line[start:i]
					state = stateFlagValueStart
					start = i + 1
				} else {
					return name, envs, flags, parseBuildConfigError{0, i + off, fmt.Sprintf("invalid character %q in flag name", r)}
				}
			case stateFlagValueStart:
				if r == '"' {
					state = stateFlagValueQuoted
					start = i + 1
				} else if r != ' ' {
					state = stateFlagValue
				} else {
					// empty value
					flags = append(flags, [2]string{valL, ""})
					state = stateStart
				}
			case stateFlagValueQuoted:
				if r == '"' {
					// end of value
					flags = append(flags, [2]string{valL, line[start:i]})
					state = stateStart
				}
			case stateFlagValue:
				if r == ' ' {
					// end of value
					flags = append(flags, [2]string{valL, line[start:i]})
					state = stateStart
				}
			default:
				panic(state)
			}
		}

		switch state {
		case stateStart:
			// nothing to do
		case stateEnvName:
			return name, envs, flags, parseBuildConfigError{0, len(line) + off, "unexpected end of line"}
		case stateEnvValueStart:
			fmt.Println("empty env value")
		case stateEnvValueQuoted:
			return name, envs, flags, parseBuildConfigError{0, len(line) + off, "unexpected end of line"}
		case stateEnvValue:
			envs = append(envs, [2]string{valL, line[start:]})
		case stateFlagNameStart:
			return name, envs, flags, parseBuildConfigError{0, len(line) + off, "unexpected end of line"}
		case stateFlagName:
			flags = append(flags, [2]string{line[start:], ""})
		case stateFlagValueStart:
			flags = append(flags, [2]string{valL, ""})
		case stateFlagValueQuoted:
			return name, envs, flags, parseBuildConfigError{0, len(line) + off, "unexpected end of line"}
		case stateFlagValue:
			flags = append(flags, [2]string{valL, line[start:]})
		default:
			panic(state)
		}
	}

	for _, r := range name {
		if !(r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)) {
			return "", nil, nil, fmt.Errorf("invalid build name %q", name)
		}
	}
	return name, envs, flags, nil
}
