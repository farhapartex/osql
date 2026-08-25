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

type NewKind int

const (
	NewFile NewKind = iota
	NewFolder
)

var newKindNames = map[NewKind]string{
	NewFile:   "file",
	NewFolder: "folder",
}

func (k NewKind) String() string {
	if name, ok := newKindNames[k]; ok {
		return name
	}
	return "unknown"
}

func ParseNewKind(s string) (NewKind, bool) {
	for kind, name := range newKindNames {
		if name == s {
			return kind, true
		}
	}
	return NewFile, false
}

type Statement struct {
	Verb       string
	Target     Target
	Path       string
	Recursive  bool
	Predicates []Predicate
	Kind       NewKind
	Data       string
	HasData    bool
}
