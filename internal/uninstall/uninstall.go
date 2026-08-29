package uninstall

import (
	"path/filepath"
	"strings"

	"github.com/farhapartex/osql/internal/oerr"
)

type Target struct {
	Path string
	Size int64
}

type Plan struct {
	Binary       Target
	Data         Target
	IncludesData bool
}

func (p Plan) TotalSize() int64 {
	if p.IncludesData {
		return p.Binary.Size + p.Data.Size
	}
	return p.Binary.Size
}

type Options struct {
	Files        Files
	LocateBinary func() (string, error)
	StateRoot    string
}

type Uninstaller struct {
	files        Files
	locateBinary func() (string, error)
	stateRoot    string
}

func New(opts Options) *Uninstaller {
	return &Uninstaller{
		files:        opts.Files,
		locateBinary: opts.LocateBinary,
		stateRoot:    opts.StateRoot,
	}
}

func (u *Uninstaller) Plan(keepData bool) (Plan, error) {
	binary, err := u.planBinary()
	if err != nil {
		return Plan{}, err
	}

	plan := Plan{Binary: binary}
	if !u.stateFolderExists() {
		return plan, nil
	}

	plan.Data = Target{Path: u.stateRoot}
	if keepData {
		return plan, nil
	}

	size, err := u.measureStateFolder()
	if err != nil {
		return Plan{}, err
	}
	plan.Data.Size = size
	plan.IncludesData = true
	return plan, nil
}

func (u *Uninstaller) Commit(plan Plan) error {
	if plan.IncludesData {
		if err := u.files.RemoveTree(plan.Data.Path); err != nil {
			return oerr.CannotRemoveData(plan.Data.Path, oerr.Reason(err))
		}
	}
	if err := u.files.Remove(plan.Binary.Path); err != nil {
		return oerr.CannotRemoveBinary(plan.Binary.Path)
	}
	return nil
}

func (u *Uninstaller) planBinary() (Target, error) {
	path, err := u.locateBinary()
	if err != nil || strings.TrimSpace(path) == "" {
		return Target{}, oerr.BinaryNotFound()
	}

	if manager, managed := managerOwning(path); managed {
		return Target{}, oerr.InstalledByPackageManager(manager.name, manager.command)
	}

	info, err := u.files.Stat(path)
	if err != nil {
		return Target{}, oerr.BinaryNotFound()
	}

	if !u.files.CanWriteInto(filepath.Dir(path)) {
		return Target{}, oerr.CannotRemoveBinary(path)
	}

	return Target{Path: path, Size: info.Size()}, nil
}

func (u *Uninstaller) stateFolderExists() bool {
	if u.stateRoot == "" {
		return false
	}
	_, err := u.files.Stat(u.stateRoot)
	return err == nil
}

func (u *Uninstaller) measureStateFolder() (int64, error) {
	if !u.files.CanWriteInto(u.stateRoot) || !u.files.CanWriteInto(filepath.Dir(u.stateRoot)) {
		return 0, oerr.CannotRemoveData(u.stateRoot, "permission denied")
	}

	size, err := u.files.DirectorySize(u.stateRoot)
	if err != nil {
		return 0, oerr.CannotRemoveData(u.stateRoot, oerr.Reason(err))
	}
	return size, nil
}

type packageManager struct {
	marker  string
	name    string
	command string
}

var packageManagers = []packageManager{
	{marker: "/Cellar/", name: "Homebrew", command: "brew uninstall osql"},
	{marker: "/nix/store/", name: "Nix", command: "nix profile remove osql"},
}

func managerOwning(path string) (packageManager, bool) {
	for _, manager := range packageManagers {
		if strings.Contains(path, manager.marker) {
			return manager, true
		}
	}
	return packageManager{}, false
}
