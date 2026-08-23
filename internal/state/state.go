package state

type Store interface {
	Ensure() error
	WriteSystemInfo(force bool) error
	History() (History, error)
}

type History interface {
	Append(line string) error
	Lines(limit int) ([]string, error)
	Clear() error
	Close() error
}
