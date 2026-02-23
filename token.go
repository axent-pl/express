package express

type tokenKind int

const (
	tokenLiteral tokenKind = iota
	tokenPlaceholder
)

type pathSegment struct {
	key     string
	index   int
	isIndex bool
}

type token struct {
	kind       tokenKind
	literal    string
	segments   []pathSegment
	hasDefault bool
	def        string
	raw        string
}
