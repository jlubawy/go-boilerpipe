package boilerpipe

// check if the tag is self-closing tag
func CheckIsTolerateTag(lookup string) bool {
	switch lookup {
	case
		"area", "base", "br", "embed", "hr", "iframe", "img", "input", "link", "meta", "param", "source", "track":
		return true
	}
	return false
}