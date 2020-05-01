package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"context"
	"flag"
	"fmt"
	bfs "github.com/demonoid81/broccolicompile/fs"
	"log"
	"os"
	"path/filepath"
	"time"
)

var progName = filepath.Base(os.Args[0])

func usage() {
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", progName)
	flag.PrintDefaults()
}

func main() {
	//log.SetFlags(0)
	//log.SetPrefix(progName + ": ")git commit -m "first commit"
	//git remote add origin git@github.com:demonoid81/broccolicompile.git
	//git push -u origin master
	//
	//flag.Usage = usage
	//flag.Parse()
	//
	//if flag.NArg() != 1 {
	//	usage()
	//	os.Exit(2)
	//}
	if err := mount("/mnt"); err != nil {
		log.Fatal(err)
	}
}

type FS struct {
	files  *bfs.Broccoli
}

var _ fs.FS = (*FS)(nil)

type Dir struct {
	files *bfs.Broccoli
	file *bfs.File
}

func (f *FS) Root() (fs.Node, error) {
	n := &Dir{
		files: f.files,
	}
	return n, nil
}

var _ fs.Node = (*Dir)(nil)

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	if d.file == nil {
		// root directory
		a.Mode = os.ModeDir | 0755
		return nil
	}
	fileAttr(d.file, a)
	return nil
}

func fileAttr(f *bfs.File, a *fuse.Attr) {
	a.Size = uint64(f.Fsize)
	a.Mode =  0644
	a.Mtime = time.Unix(f.Ftime, 0)
	a.Ctime = time.Unix(f.Ftime, 0)
	a.Crtime = time.Unix(f.Ftime, 0)
}

func mount(mountpoint string) error {

	c, err := fuse.Mount(mountpoint)
	if err != nil {
		return err
	}
	defer c.Close()

	filesystem := &FS{
		files: broccoli,
	}
	if err := fs.Serve(c, filesystem); err != nil {
		return err
	}

	// проверяем ошибки при монтировании
	<-c.Ready
	if err := c.MountError; err != nil {
		return err
	}

	return nil
}

var _ = fs.NodeRequestLookuper(&Dir{})


func (d *Dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	path := req.Name
	if d.file != nil {
		path = d.file.Name() + path
	}
	fmt.Println(path)
	for _, f := range d.files.Files {
		var name = f.Name()
		switch {
		case name == path:
			child := &File{
				file: f,
			}
			return child, nil
		case name[:len(name)-1] == path && name[len(name)-1] == '/':
			child := &Dir{
				files: d.files,
				file:    f,
			}
			return child, nil
		}
	}
	return nil, fuse.ENOENT
}

var _ = fs.HandleReadDirAller(&Dir{})

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	//prefix := ""
	//if d.file != nil {
	//	prefix = d.file.Name()
	//}

	var res []fuse.Dirent
	//for _, f := range d.files.Files {
	//	if !strings.HasPrefix(f.Name(), prefix) {
	//		continue
	//	}
	//	name := f.Name()[len(prefix):]
	//	if name == "" {
	//		// the dir itself, not a child
	//		continue
	//	}
	//	if strings.ContainsRune(name[:len(name)-1], '/') {
	//		// contains slash in the middle -> is in a deeper subdir
	//		continue
	//	}
	//	var de fuse.Dirent
	//	if name[len(name)-1] == '/' {
	//		// directory
	//		name = name[:len(name)-1]
	//		de.Type = fuse.DT_Dir
	//	}
	//	de.Name = name
	//	res = append(res, de)
	//}
	return res, nil
}

type File struct {
	file *bfs.File
}

var _ fs.Node = (*File)(nil)

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	fileAttr(f.file, a)
	return nil
}
//
//var _ = fs.NodeOpener(&File{})
//
//func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
//	r, err := f.file.Open()
//	if err != nil {
//		return nil, err
//	}
//	//// individual entries inside a zip file are not seekable
//	resp.Flags |= fuse.OpenNonSeekable
//	return &FileHandle{r: r}, nil
//}
//
//type FileHandle struct {
//	r io.ReadCloser
//}
//
//var _ fs.Handle = (*FileHandle)(nil)
//
//var _ fs.HandleReleaser = (*FileHandle)(nil)
//
//func (fh *FileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
//	return fh.r.Close()
//}
//
//var _ = fs.HandleReader(&FileHandle{})
//
//func (fh *FileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
//	buf := make([]byte, req.Size)
//	n, err := io.ReadFull(fh.r, buf)
//	if err == io.ErrUnexpectedEOF || err == io.EOF {
//		err = nil
//	}
//	resp.Data = buf[:n]
//	return err
//}