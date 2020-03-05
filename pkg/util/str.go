package util

func SearchVersion(a []string, b string) bool {
	if len(a) == 0 {
		return false
	}

	for _, v := range a {
		if v == b {
			return true
		}
	}

	return false
}
