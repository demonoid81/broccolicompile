package main

import (
	"aletheia.icu/broccoli/data"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"bazil.org/fuse/fuseutil"
	"context"
	"flag"
	"fmt"
	bfs "github.com/demonoid81/broccolicompile/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

var progName = filepath.Base(os.Args[0])

func usage() {
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", progName)
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix(progName + ": ")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}

	go func() {
		var appTime time.Time
		interrupt := make(chan os.Signal,1)
		signal.Notify(interrupt, os.Interrupt)

		done := make(chan struct{})

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <- ticker.C:
				if appTime.Before(time.Now()) {
					fuse.Unmount(flag.Arg(0))
					select {
					case <-done:
					case <-time.After(time.Second):
					}
					return
				}
				appTime = time.Now()
			case <- interrupt:
				log.Println("interput")
				fuse.Unmount(flag.Arg(0))
				select {
				case <-done:
				case <-time.After(time.Second):
				}
				return
			}
		}
	}()

	if err := mount(flag.Arg(0)); err != nil {
		log.Fatal(err)
	}

}

type FS struct {
	files  *bfs.Broccoli
}

var _ fs.FS = (*FS)(nil)

type Dir struct {
	files *bfs.Broccoli
	file  *bfs.File
}

func (f *FS) Root() (fs.Node, error) {
	n := &Dir{
		files: f.files,
	}
	return n, nil
}

var _ fs.Node = (*Dir)(nil)

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0755
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
		files: data.Broccoli,
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
		path = d.file.Fpath + "/" + path
	}
	for _, f := range d.files.Files {
		var name = f.Fpath
		switch {
		case !f.IsDir() && name == path:
			child := &File{
				file: f,
			}
			return child, nil
		case f.IsDir() && name == path:
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
	prefix := ""
	if d.file != nil {
		prefix = d.file.Fpath
	}
	var res []fuse.Dirent
	for _, f := range d.files.Files {
		if !strings.HasPrefix(f.Fpath, prefix) {
			continue
		}

		p1arr := strings.Split(f.Fpath, "/")
		p2arr := strings.Split(prefix, "/")

		if len(p1arr)-len(p2arr) == 1 && len(prefix) > 0 {
			var de fuse.Dirent
			if f.IsDir() {
				de.Type = fuse.DT_Dir
			} else {
				de.Type = fuse.DT_File
			}
			de.Name = f.Fname
			res = append(res, de)
		} else if prefix == "" && len(p1arr) == 1 {
			var de fuse.Dirent
			if f.IsDir() {
				de.Type = fuse.DT_Dir
			} else {
				de.Type = fuse.DT_File
			}
			de.Name = f.Fname
			res = append(res, de)
		}
	}
	return res, nil
}

type File struct {
	file *bfs.File
}

var _ fs.Node = (*File)(nil)

var _ fs.Handle = (*File)(nil)

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	fileAttr(f.file, a)
	return nil
}

var _ = fs.NodeOpener(&File{})

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	err := f.file.Open()
	if err != nil {
		return nil, err
	}
	//// individual entries inside a zip file are not seekable
	resp.Flags |= fuse.OpenNonSeekable
	return f, nil
}

var _ fs.HandleReleaser = (*File)(nil)

func (f *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return nil
}

var _ fs.HandleReader = (*File)(nil)

func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	fuseutil.HandleRead(req,resp,f.file.Data)
	return nil
}