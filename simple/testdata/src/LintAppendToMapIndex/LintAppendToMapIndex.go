package pkg

func fn() {
	var m = map[string][]string{}

	if _, ok := m["k1"]; ok { // want `unnecessary guard around call to append`
		m["k1"] = append(m["k1"], "v1", "v2")
	} else {
		m["k1"] = []string{"v1", "v2"}
	}

	if _, ok := m["k1"]; ok {
		m["k1"] = append(m["k1"], "v1", "v2")
	} else {
		m["k1"] = []string{"v1"}
	}

	if _, ok := m["k1"]; ok {
		m["k2"] = append(m["k2"], "v1")
	} else {
		m["k1"] = []string{"v1"}
	}

	k1 := "key"
	if _, ok := m[k1]; ok { // want `unnecessary guard around call to append`
		m[k1] = append(m[k1], "v1", "v2")
	} else {
		m[k1] = []string{"v1", "v2"}
	}

	// ellipsis is not currently supported
	v := []string{"v1", "v2"}
	if _, ok := m["k1"]; ok {
		m["k1"] = append(m["k1"], v...)
	} else {
		m["k1"] = v
	}
}
