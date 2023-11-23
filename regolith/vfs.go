package regolith

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/stirante/dokan-go"
	_ "github.com/stirante/dokan-go/dokan_header"
	"github.com/stirante/dokan-go/winacl"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Root struct {
	Source    string
	MountName string
}

type VirtualFileSystem struct {
	files     map[string]*VirtualFile
	lock      sync.Mutex
	MountPath string
	Handle    *dokan.MountHandle
	Roots     []Root
}

func CreateTestFS() {
	// When testing, make sure all these folders exist

	// Create new FS. It will use C:\regolith_test_source as source directory and C:\regolith_test_1 as mount point
	fs, err := NewRegolithFS([]Root{
		{
			Source:    "C:\\regolith_test_source",
			MountName: "source",
		},
	}, "C:\\regolith_test")
	// Check for errors. When we add this to an actual code, we can fall back to a different method
	if err != nil {
		//fmt.Println(err)
		return
	}
	// MountPath can change, but usually it will be the same as the one we provided
	//fmt.Printf("Mounted %s at %s\n", fs.Source, fs.MountPath)
	// Wait for input to unmount
	Logger.Info("Press enter to stop")
	fmt.Scanln()

	// Unmount the FS
	err2 := fs.Handle.Close()
	if err2 != nil {
		//fmt.Println(err2)
	}
	// Save the results to C:\regolith_test_target
	err2 = fs.SaveToPath("source", "C:\\regolith_test_target", false, false)
	if err2 != nil {
		//fmt.Println(err2)
	}
}

func NewRegolithFS(roots []Root, mountPath string) (*VirtualFileSystem, error) {
	stat, err := os.Stat(mountPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(mountPath, 0755)
			if err != nil {
				return nil, burrito.WrapErrorf(err, osMkdirError, mountPath)
			}
		} else {
			return nil, burrito.WrapErrorf(err, osStatErrorAny, mountPath)
		}
	} else if !stat.IsDir() {
		return nil, burrito.WrappedErrorf(isDirNotADirError, mountPath)
	}
	fs := &VirtualFileSystem{
		Roots: roots,
		files: make(map[string]*VirtualFile),
	}
	config := dokan.Config{
		MountFlags: dokan.MountManager, // | dokan.CDebug | dokan.CStderr,
		Path:       mountPath,
		FileSystem: fs,
	}
	mount, err := dokan.Mount(&config)
	if err != nil {
		return nil, burrito.WrapErrorf(err, "Failed to mount at %s", mountPath)
	}
	fs.Handle = mount
	fs.MountPath = mount.Dir
	// Create shells for all roots
	for _, root := range roots {
		file, err := mirrorRegolithFile(fs, root.Source, nil)
		if err != nil {
			err2 := mount.Close()
			if err2 != nil {
				Logger.Warnf("Failed to close VFS handle: %s", err2.Error())
			}
			return nil, burrito.WrapErrorf(err, "Failed to mirror %s", root.Source)
		}
		root.Source = filepath.Clean(root.Source)
		fs.files["\\"+root.MountName] = file
		err = filepath.Walk(root.Source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return burrito.WrapErrorf(err, "Failed to walk %s", path)
			}
			if path == root.Source {
				return nil
			}

			relativePath := strings.TrimPrefix(path, root.Source)
			fs.files["\\"+root.MountName+relativePath], err = mirrorRegolithFile(fs, path, info)
			if err != nil {
				return burrito.WrapErrorf(err, "Failed to mirror %s", path)
			}
			return nil
		})
		if err != nil {
			err2 := mount.Close()
			if err2 != nil {
				Logger.Warnf("Failed to close VFS handle: %s", err2.Error())
			}
			return nil, burrito.WrapErrorf(err, "Failed to walk %s", root.Source)
		}
	}
	return fs, nil
}

func (fs *VirtualFileSystem) WithContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctx, nil
}
func (fs *VirtualFileSystem) CreateFile(ctx context.Context, fi *dokan.FileInfo, data *dokan.CreateData) (file dokan.File, status dokan.CreateStatus, err error) {
	//marshal, _ := json.Marshal(data)
	//fmt.Printf("CreateFile: %s %s\n", fi.Path(), marshal)

	// Check if the file already exists in our map
	fs.lock.Lock()
	defer fs.lock.Unlock()
	path := fi.Path()
	// Handle case when listing the roots
	if path == "\\" {
		if data.CreateDisposition == dokan.FileCreate {
			return nil, dokan.CreateStatus(0), burrito.WrappedErrorf("File already exists at %s", path)
		}
		//fmt.Printf("Root directory at %s\n", fi.Path())
		return newRootFile(fs), dokan.ExistingDir, data.ReturningDirAllowed()
	}
	if f, ok := fs.files[fi.Path()]; ok {
		if data.CreateDisposition == dokan.FileCreate {
			//fmt.Printf("File already exists at %s\n", fi.Path())
			return nil, dokan.CreateStatus(0), burrito.WrappedErrorf("File already exists at %s", path)
		}
		if f.IsDir {
			//fmt.Printf("Cached directory at %s\n", fi.Path())
			return f, dokan.ExistingDir, data.ReturningDirAllowed()
		}
		//fmt.Printf("Cached file at %s\n", fi.Path())
		return f, dokan.ExistingFile, data.ReturningFileAllowed()
	}

	// TODO: Do we still need it, if we create all shells when creating FS?

	// Check if the file exists in the source directory
	list := strings.Split(path, "\\")
	// Remove empty first elements
	if len(list) > 0 && list[0] == "" {
		list = list[1:]
	}
	if len(list) == 0 {
		//fmt.Printf("Invalid path %s\n", fi.Path())
		return nil, dokan.CreateStatus(0), burrito.WrappedErrorf("Invalid path %s", path)
	}
	// Find root
	var root Root
	for _, r := range fs.Roots {
		if r.MountName == list[0] {
			root = r
			break
		}
	}
	// If no root found, return error
	if root.Source == "" {
		//fmt.Printf("Invalid root %s\n", list[0])
		return nil, dokan.CreateStatus(0), burrito.WrappedErrorf("Invalid path %s", path)
	}
	list[0] = root.Source
	sourcePath := filepath.Join(list...)
	//fmt.Printf("Source path: %s\n", sourcePath)
	stat, err := os.Stat(sourcePath)
	if err == nil {
		if data.CreateDisposition == dokan.FileCreate {
			//fmt.Printf("File already exists at %s\n", fi.Path())
			return nil, dokan.CreateStatus(0), burrito.WrappedErrorf("File already exists at %s", path)
		}
		result, err := mirrorRegolithFile(fs, sourcePath, stat)
		if err != nil {
			//fmt.Printf("Failed to mirror %s: %s\n", sourcePath, err.Error())
			return nil, dokan.CreateStatus(0), burrito.PassError(err)
		}
		fs.files[fi.Path()] = result
		if result.IsDir {
			//fmt.Printf("Directory at %s\n", fi.Path())
			return result, dokan.ExistingDir, data.ReturningDirAllowed()
		}
		//fmt.Printf("File at %s\n", fi.Path())
		return result, dokan.ExistingFile, data.ReturningFileAllowed()
	} else if !os.IsNotExist(err) {
		//fmt.Printf("Failed to mirror %s: %s\n", sourcePath, err.Error())
		return nil, dokan.CreateStatus(0), burrito.WrapErrorf(err, "Failed to stat %s", sourcePath)
	}

	if data.CreateDisposition == dokan.FileCreate {
		// Otherwise, create a new RAMFile
		ramFile := newRegolithFile(fs)
		ramFile.IsDir = (data.CreateOptions & dokan.FileDirectoryFile) != 0
		fs.files[path] = ramFile

		if ramFile.IsDir {
			//fmt.Printf("New directory at %s\n", fi.Path())
			return ramFile, dokan.NewDir, data.ReturningDirAllowed()
		}
		//fmt.Printf("New file at %s\n", fi.Path())
		return ramFile, dokan.NewFile, data.ReturningFileAllowed()
	}

	//fmt.Printf("File does not exist at %s\n", fi.Path())
	return nil, dokan.CreateStatus(0), burrito.WrappedErrorf("File does not exist at %s", path)
}

func (fs *VirtualFileSystem) GetDiskFreeSpace(ctx context.Context) (dokan.FreeSpace, error) {
	return dokan.FreeSpace{
		FreeBytesAvailable:     1024 * 1024 * 1024,
		TotalNumberOfBytes:     1024 * 1024 * 1024,
		TotalNumberOfFreeBytes: 1024 * 1024 * 1024,
	}, nil
}

func (fs *VirtualFileSystem) GetVolumeInformation(ctx context.Context) (dokan.VolumeInformation, error) {
	return dokan.VolumeInformation{
		VolumeName:             "REGOLITH",
		VolumeSerialNumber:     0,
		MaximumComponentLength: 1024,
		FileSystemFlags:        0,
		FileSystemName:         "REGOLITH",
	}, nil
}

func (fs *VirtualFileSystem) MoveFile(ctx context.Context, sourceHandle dokan.File, sourceFileInfo *dokan.FileInfo, targetPath string, replaceExisting bool) error {
	//fmt.Printf("MoveFile: %s -> %s\n", sourceFileInfo.Path(), targetPath)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	// Move the old file to new path
	if oldFile, ok := fs.files[sourceFileInfo.Path()]; ok {
		if _, ok := fs.files[targetPath]; ok {
			if !replaceExisting {
				return burrito.WrappedErrorf("File already exists at %s", targetPath)
			}
		}
		fs.files[targetPath] = oldFile
	} else {
		return burrito.WrappedErrorf("File does not exist at %s", sourceFileInfo.Path())
	}

	return nil
}

func (fs *VirtualFileSystem) SaveToPath(from, to string, makeReadOnly, copyParentAcl bool) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	// TODO: honor makeReadOnly and copyParentAcl
	for virtualPath, file := range fs.files {
		if !file.Deleted && !file.IsDir && strings.HasPrefix(virtualPath, "\\"+from) {
			realPath := filepath.Join(to, strings.TrimPrefix(virtualPath, "\\"+from))
			dir := filepath.Dir(realPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			err := file.EnsureLoaded()
			if err != nil {
				return burrito.PassError(err)
			}
			if err := os.WriteFile(realPath, file.contents, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func (fs *VirtualFileSystem) SyncToPath(from, to string, makeReadOnly, copyParentAcl bool) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	// TODO: honor makeReadOnly and copyParentAcl
	for virtualPath, file := range fs.files {
		if !file.Deleted && !file.IsDir && strings.HasPrefix(virtualPath, "\\"+from) {
			realPath := filepath.Join(to, strings.TrimPrefix(virtualPath, "\\"+from))
			stat, err := os.Stat(realPath)
			if err != nil && !os.IsNotExist(err) {
				return burrito.WrapErrorf(err, osStatErrorAny, realPath)
			}
			if os.IsNotExist(err) || stat.Size() != file.Size || stat.ModTime() != file.LastWriteTime {
				dir := filepath.Dir(realPath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				err := file.EnsureLoaded()
				if err != nil {
					return burrito.PassError(err)
				}
				if err := os.WriteFile(realPath, file.contents, 0644); err != nil {
					return err
				}
			}
		}
	}
	// Remove files/folders in destination that are not in source
	err := filepath.Walk(to, func(destPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(to, destPath)
		if err != nil {
			return burrito.WrapErrorf(err, filepathRelError, to, destPath)
		}
		if relPath == "." {
			return nil
		}
		f, ok := fs.files["\\"+from+"\\"+relPath]
		if !ok || f.Deleted {
			Logger.Debugf("SYNC: Removing file %s", destPath)
			return os.RemoveAll(destPath)
		}
		return nil
	})

	if err != nil {
		return burrito.WrapErrorf(err, osRemoveError, to)
	}
	return nil
}

func (fs *VirtualFileSystem) ErrorPrint(err error) {
	//fmt.Printf("ErrorPrint: %s\n", err.Error())
}

func (fs *VirtualFileSystem) Printf(format string, v ...interface{}) {
	//fmt.Printf(format, v...)
}

func (fs *VirtualFileSystem) Close() {
	err := fs.Handle.Close()
	if err != nil {
		Logger.Warnf("Failed to close VFS handle: %s", err.Error())
	}
}

type VirtualFile struct {
	lock          sync.Mutex
	CreationTime  time.Time
	LastReadTime  time.Time
	LastWriteTime time.Time
	Size          int64
	contents      []byte
	Deleted       bool
	loaded        bool
	Source        string
	IsDir         bool
	fs            *VirtualFileSystem
}

func (r *VirtualFile) GetContents() ([]byte, error) {
	err := r.EnsureLoaded()
	if err != nil {
		return nil, burrito.PassError(err)
	}
	return r.contents, nil
}

func (r *VirtualFile) FlushFileBuffers(ctx context.Context, fi *dokan.FileInfo) error {
	//fmt.Println("VirtualFile.FlushFileBuffers")
	return nil
}

func (r *VirtualFile) GetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	//fmt.Println("VirtualFile.GetFileSecurity")
	return nil
}

func (r *VirtualFile) SetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	//fmt.Println("VirtualFile.SetFileSecurity")
	return nil
}

func (r *VirtualFile) CanDeleteFile(ctx context.Context, fi *dokan.FileInfo) error {
	//fmt.Println("VirtualFile.CanDeleteFile")
	return nil
}

func (r *VirtualFile) CanDeleteDirectory(ctx context.Context, fi *dokan.FileInfo) error {
	//fmt.Println("VirtualFile.CanDeleteDirectory")
	return nil
}

func (r *VirtualFile) Cleanup(ctx context.Context, fi *dokan.FileInfo) {
	//fmt.Printf("VirtualFile.Cleanup %s\n", fi.Path())
	if fi.IsDeleteOnClose() {
		r.lock.Lock()
		r.Deleted = true
		r.contents = nil
		r.Size = 0
		r.lock.Unlock()
	}
}

func (r *VirtualFile) CloseFile(ctx context.Context, fi *dokan.FileInfo) {
	//fmt.Printf("VirtualFile.CloseFile %s\n", fi.Path())
}

func newRegolithFile(fs *VirtualFileSystem) *VirtualFile {
	var r VirtualFile
	r.CreationTime = time.Now()
	r.LastReadTime = r.CreationTime
	r.LastWriteTime = r.CreationTime
	r.fs = fs
	r.Deleted = false
	r.loaded = true
	r.Size = 0
	r.contents = []byte{}
	return &r
}

func mirrorRegolithFile(fs *VirtualFileSystem, source string, stat os.FileInfo) (*VirtualFile, error) {
	var r VirtualFile
	r.fs = fs
	r.Source = source
	if stat == nil {
		s, err := os.Stat(source)
		if err != nil {
			return nil, burrito.WrapErrorf(err, "Failed to stat %s", source)
		}
		stat = s
	}
	r.IsDir = stat.IsDir()
	r.CreationTime = stat.ModTime()
	r.LastReadTime = r.CreationTime
	r.LastWriteTime = r.CreationTime
	r.Deleted = false
	r.loaded = false
	r.Size = stat.Size()
	return &r, nil
}

func (r *VirtualFile) FindFiles(ctx context.Context, fi *dokan.FileInfo, search string, fill func(*dokan.NamedStat) error) error {
	//fmt.Printf("VirtualFile.FindFiles %s\n", fi.Path())
	//fmt.Printf("sourcePath: %s\n", r.source)
	entries, err := os.ReadDir(r.Source)
	if err != nil {
		return burrito.WrapErrorf(err, "Failed to read directory %s", r.Source)
	}
	r.fs.lock.Lock()
	defer r.fs.lock.Unlock()
	for _, entry := range entries {
		//fmt.Printf("entry: %s\n", entry.Name())
		if entry.Name() == "." || entry.Name() == ".." || entry.Name() == "/" || entry.Name() == "\\" {
			continue // Skip root itself
		}
		path := filepath.Join(fi.Path(), entry.Name())
		if _, ok := r.fs.files[path]; ok {
			continue // Skip files that are already in the map
		}
		r.fs.files[path], err = mirrorRegolithFile(r.fs, filepath.Join(r.Source, entry.Name()), nil)
		if err != nil {
			return burrito.WrapErrorf(err, "Failed to mirror %s", path)
		}
	}
	for k, v := range r.fs.files {
		if !v.Deleted && filepath.Dir(k) == fi.Path() && k != "." && k != ".." && k != "/" && k != "\\" {
			if v.IsDir {
				err = fill(&dokan.NamedStat{
					Name: filepath.Base(k),
					Stat: dokan.Stat{
						FileAttributes: dokan.FileAttributeDirectory,
					},
				})
			} else {
				err = fill(&dokan.NamedStat{
					Name: filepath.Base(k),
					Stat: dokan.Stat{
						FileSize:   v.Size,
						LastAccess: v.LastReadTime,
						LastWrite:  v.LastWriteTime,
						Creation:   v.CreationTime,
					},
				})
			}
			if err != nil {
				return burrito.WrapErrorf(err, "Failed to send %s to readdir", k)
			}
		}
	}
	return nil
}

func (r *VirtualFile) SetFileAttributes(ctx context.Context, fi *dokan.FileInfo, fileAttributes dokan.FileAttribute) error {
	//fmt.Println("VirtualFile.SetFileAttributes")
	return nil
}

func (r *VirtualFile) LockFile(ctx context.Context, fi *dokan.FileInfo, offset int64, length int64) error {
	//fmt.Println("VirtualFile.LockFile")
	return nil
}
func (r *VirtualFile) UnlockFile(ctx context.Context, fi *dokan.FileInfo, offset int64, length int64) error {
	//fmt.Println("VirtualFile.UnlockFile")
	return nil
}

func (r *VirtualFile) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	//fmt.Printf("VirtualFile.GetFileInformation %s\n", fi.Path())
	r.lock.Lock()
	defer r.lock.Unlock()
	// this way we can filter out explorer, vscode and other processes, that don't matter
	// alternatively we could save pid of the filter and check it here
	//fmt.Printf("pid: %d\n", fi.ProcessId())
	//pid, err := getExecutableNameFromPID(fi.ProcessId())
	//if err != nil {
	//fmt.Printf("Failed to get executable name: %s\n", err.Error())
	//} else {
	//fmt.Printf("executable: %s\n", pid)
	//}
	attr := dokan.FileAttributeNormal
	if r.IsDir {
		attr = dokan.FileAttributeDirectory
	}
	result := dokan.Stat{
		FileSize:       r.Size,
		LastAccess:     r.LastReadTime,
		LastWrite:      r.LastWriteTime,
		Creation:       r.CreationTime,
		FileAttributes: attr,
	}
	//marshal, _ := json.Marshal(result)
	//fmt.Printf("GetFileInformation: %s %s\n", r.Source, marshal)
	return &result, nil
}

var processNameCache = make(map[uint64]string)

func getExecutableNameFromPID(pid uint64) (string, error) {
	if name, ok := processNameCache[pid]; ok {
		return name, nil
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)

	var buffer [windows.MAX_PATH]uint16
	size := uint32(len(buffer))
	err = windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size)
	if err != nil {
		return "", err
	}

	executableName := windows.UTF16ToString(buffer[:size])
	processNameCache[pid] = executableName
	return executableName, nil
}

func (r *VirtualFile) EnsureLoaded() error {
	if !r.loaded {
		if r.IsDir {
			r.loaded = true
		} else {
			if r.Source != "" {
				data, err := os.ReadFile(r.Source)
				if err != nil {
					//fmt.Printf("Failed to read file %s: %s\n", r.Source, err.Error())
					return burrito.WrapErrorf(err, "Failed to read file %s", r.Source)
				}
				r.contents = data
				r.Size = int64(len(data))
				r.loaded = true
			}
		}
	}
	return nil
}

func (r *VirtualFile) ReadFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	//fmt.Printf("VirtualFile.ReadFile %s\n", fi.Path())
	r.lock.Lock()
	defer r.lock.Unlock()
	r.LastReadTime = time.Now()
	err := r.EnsureLoaded()
	if err != nil {
		return 0, burrito.PassError(err)
	}
	rd := bytes.NewReader(r.contents)
	return rd.ReadAt(bs, offset)
}

func (r *VirtualFile) WriteFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	//fmt.Println("VirtualFile.WriteFile")
	r.lock.Lock()
	defer r.lock.Unlock()
	err := r.EnsureLoaded()
	if err != nil {
		return 0, burrito.PassError(err)
	}
	r.LastWriteTime = time.Now()
	maxl := len(r.contents)
	if int(offset)+len(bs) > maxl {
		maxl = int(offset) + len(bs)
		r.contents = append(r.contents, make([]byte, maxl-len(r.contents))...)
	}
	n := copy(r.contents[int(offset):], bs)
	r.Size = int64(maxl)
	return n, nil
}
func (r *VirtualFile) SetFileTime(ctx context.Context, fi *dokan.FileInfo, creationTime time.Time, lastReadTime time.Time, lastWriteTime time.Time) error {
	//fmt.Println("VirtualFile.SetFileTime")
	r.lock.Lock()
	defer r.lock.Unlock()
	if !lastWriteTime.IsZero() {
		r.LastWriteTime = lastWriteTime
	}
	return nil
}
func (r *VirtualFile) SetEndOfFile(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	//fmt.Println("VirtualFile.SetEndOfFile")
	r.lock.Lock()
	defer r.lock.Unlock()
	err := r.EnsureLoaded()
	if err != nil {
		return burrito.PassError(err)
	}
	r.LastWriteTime = time.Now()
	switch {
	case int(length) < len(r.contents):
		r.contents = r.contents[:int(length)]
	case int(length) > len(r.contents):
		r.contents = append(r.contents, make([]byte, int(length)-len(r.contents))...)
	}
	r.Size = length
	return nil
}
func (r *VirtualFile) SetAllocationSize(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	//fmt.Println("VirtualFile.SetAllocationSize")
	r.lock.Lock()
	defer r.lock.Unlock()
	err := r.EnsureLoaded()
	if err != nil {
		return burrito.PassError(err)
	}
	r.LastWriteTime = time.Now()
	switch {
	case int(length) < len(r.contents):
		r.contents = r.contents[:int(length)]
		r.Size = length
	}
	return nil
}

type RootFile struct {
	lock          sync.Mutex
	CreationTime  time.Time
	LastReadTime  time.Time
	LastWriteTime time.Time
	fs            *VirtualFileSystem
}

func (r *RootFile) FlushFileBuffers(ctx context.Context, fi *dokan.FileInfo) error {
	//fmt.Printf("FlushFileBuffers: %s\n", fi.Path())
	return nil
}

func (r *RootFile) GetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	//fmt.Printf("GetFileSecurity: %s\n", fi.Path())
	return nil
}

func (r *RootFile) SetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	//fmt.Printf("SetFileSecurity: %s\n", fi.Path())
	return nil
}

func (r *RootFile) CanDeleteFile(ctx context.Context, fi *dokan.FileInfo) error {
	//fmt.Printf("CanDeleteFile: %s\n", fi.Path())
	return burrito.WrappedErrorf("Unsupported operation on root file")
}

func (r *RootFile) CanDeleteDirectory(ctx context.Context, fi *dokan.FileInfo) error {
	//fmt.Printf("CanDeleteDirectory: %s\n", fi.Path())
	return burrito.WrappedErrorf("Unsupported operation on root file")
}

func (r *RootFile) Cleanup(ctx context.Context, fi *dokan.FileInfo) {
	//fmt.Printf("Cleanup: %s\n", fi.Path())
}

func (r *RootFile) CloseFile(ctx context.Context, fi *dokan.FileInfo) {
	//fmt.Printf("CloseFile: %s\n", fi.Path())
}

func newRootFile(fs *VirtualFileSystem) *RootFile {
	var r RootFile
	r.CreationTime = time.Now()
	r.LastReadTime = r.CreationTime
	r.LastWriteTime = r.CreationTime
	r.fs = fs
	return &r
}

func (r *RootFile) FindFiles(ctx context.Context, fi *dokan.FileInfo, search string, fill func(*dokan.NamedStat) error) error {
	//fmt.Printf("FindFiles: %s\n", fi.Path())
	for _, root := range r.fs.Roots {
		//fmt.Printf("root: %s\n", root.MountName)
		err := fill(&dokan.NamedStat{
			Name: root.MountName,
			Stat: dokan.Stat{
				FileAttributes: dokan.FileAttributeDirectory,
			},
		})
		if err != nil {
			return burrito.PassError(err)
		}
	}
	return nil
}

func (r *RootFile) SetFileAttributes(ctx context.Context, fi *dokan.FileInfo, fileAttributes dokan.FileAttribute) error {
	//fmt.Printf("SetFileAttributes: %s\n", fi.Path())
	return nil
}

func (r *RootFile) LockFile(ctx context.Context, fi *dokan.FileInfo, offset int64, length int64) error {
	//fmt.Printf("LockFile: %s\n", fi.Path())
	return nil
}
func (r *RootFile) UnlockFile(ctx context.Context, fi *dokan.FileInfo, offset int64, length int64) error {
	//fmt.Printf("UnlockFile: %s\n", fi.Path())
	return nil
}

func (r *RootFile) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	//fmt.Printf("GetFileInformation: %s\n", fi.Path())
	r.lock.Lock()
	defer r.lock.Unlock()
	result := dokan.Stat{
		LastAccess:     r.LastReadTime,
		LastWrite:      r.LastWriteTime,
		Creation:       r.CreationTime,
		FileAttributes: dokan.FileAttributeDirectory,
	}
	return &result, nil
}

func (r *RootFile) ReadFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	//fmt.Printf("ReadFile: %s\n", fi.Path())
	return 0, burrito.WrappedErrorf("Unsupported operation on root file")
}

func (r *RootFile) WriteFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	//fmt.Printf("WriteFile: %s\n", fi.Path())
	return 0, burrito.WrappedErrorf("Unsupported operation on root file")
}
func (r *RootFile) SetFileTime(ctx context.Context, fi *dokan.FileInfo, creationTime time.Time, lastReadTime time.Time, lastWriteTime time.Time) error {
	//fmt.Printf("SetFileTime: %s\n", fi.Path())
	r.lock.Lock()
	defer r.lock.Unlock()
	if !lastWriteTime.IsZero() {
		r.LastWriteTime = lastWriteTime
	}
	return nil
}
func (r *RootFile) SetEndOfFile(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	//fmt.Printf("SetEndOfFile: %s\n", fi.Path())
	return burrito.WrappedErrorf("Unsupported operation on root file")
}
func (r *RootFile) SetAllocationSize(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	//fmt.Printf("SetAllocationSize: %s\n", fi.Path())
	return burrito.WrappedErrorf("Unsupported operation on root file")
}
