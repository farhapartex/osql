package query

type Target int

const (
	TargetAll Target = iota
	TargetFiles
	TargetFolders
)

var targetNames = map[Target]string{
	TargetAll:     "all",
	TargetFiles:   "files",
	TargetFolders: "folders",
}

func (t Target) String() string {
	if name, ok := targetNames[t]; ok {
		return name
	}
	return "unknown"
}

func ParseTarget(s string) (Target, bool) {
	for target, name := range targetNames {
		if name == s {
			return target, true
		}
	}
	return TargetAll, false
}

type Predicate struct {
	Field string
	Op    string
	Value string
}

type Statement struct {
	Verb       string
	Target     Target
	Path       string
	Recursive  bool
	Predicates []Predicate
}
