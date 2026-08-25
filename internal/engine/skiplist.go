package engine

type SkipList struct {
	names map[string]struct{}
	paths map[string]struct{}
}

func NewSkipList(names, paths []string) SkipList {
	s := SkipList{
		names: make(map[string]struct{}, len(names)),
		paths: make(map[string]struct{}, len(paths)),
	}
	for _, n := range names {
		s.names[n] = struct{}{}
	}
	for _, p := range paths {
		s.paths[p] = struct{}{}
	}
	return s
}

func DefaultSkipList() SkipList {
	return NewSkipList(
		[]string{".git", "node_modules", ".Trash", ".Spotlight-V100", ".fseventsd"},
		[]string{"Library/Caches", "Library/Containers", "System", "Volumes", "proc", "sys", "dev", "private/var/vm"},
	)
}

func EmptySkipList() SkipList {
	return NewSkipList(nil, nil)
}

func (s SkipList) SkipsName(name string) bool {
	_, ok := s.names[name]
	return ok
}

func (s SkipList) SkipsPath(fsPath string) bool {
	_, ok := s.paths[fsPath]
	return ok
}

func (s SkipList) Skips(fsPath, name string) bool {
	return s.SkipsName(name) || s.SkipsPath(fsPath)
}

func (s SkipList) Names() int {
	return len(s.names)
}
