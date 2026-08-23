package reader

type LineReader interface {
	ReadLine(prompt string) (string, error)
	AddHistory(line string)
	Close() error
}
