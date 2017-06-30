package module

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/pdbogen/gosible/transport"
	"github.com/pdbogen/gosible/types"
	"io"
	"os"
	"strconv"
	"strings"
)

type File struct {
	source  *string
	literal *string
	dest    string
	mode    *int32
	uid     *int32
	gid     *int32
}

func check(path string) error {
	if stat, err := os.Stat(path); os.IsNotExist(err) {
		return errors.New("does not exist")
	} else if stat.IsDir() {
		return errors.New("is a directory")
	}
	return nil
}

func int32Ptr(i int32) (ret *int32) {
	ret = new(int32)
	*ret = i
	return
}

func (f *File) Configure(target *types.Target, params map[string]string) error {
	*f = File{}
	src, srcOk := params["source"]
	if srcOk {
		if src[0] != '/' {
			src = strings.TrimRight(target.Metadata["rootpath"], "/") + "/" + src
		}
		if err := check(src); err != nil {
			return fmt.Errorf("file: source %s: %s", src, err)
		}
	}

	lit, litOk := params["literal"]
	if litOk && srcOk {
		return errors.New("file configured with both source and literal")
	} else if !litOk && !srcOk {
		return errors.New("file configured with neither source nor literal")
	}

	if srcOk {
		f.source = &src
	} else {
		f.literal = &lit
	}

	if dst, ok := params["dest"]; !ok {
		return errors.New("file configured without destination")
	} else {
		f.dest = dst
	}

	if mode, ok := params["mode"]; ok {
		if n, err := strconv.ParseInt(mode, 8, 16); err != nil {
			return fmt.Errorf("parsing (non-octal?) mode %s: %s", mode, err)
		} else {
			f.mode = int32Ptr(int32(n))
		}
	}

	if uid, ok := params["uid"]; ok {
		if n, err := strconv.Atoi(uid); err != nil {
			return fmt.Errorf("parsing (non-numeric?) UID %s: %s", uid, err)
		} else {
			f.uid = int32Ptr(int32(n))
		}
	}

	if gid, ok := params["gid"]; ok {
		if n, err := strconv.Atoi(gid); err != nil {
			return fmt.Errorf("parsing (non-numeric?) GID %s: %s", gid, err)
		} else {
			f.gid = int32Ptr(int32(n))
		}
	}

	return nil
}

func (f *File) writeFile(tr transport.Transport) (bool, error) {
	var changed bool
	var src io.Reader
	if f.literal != nil {
		src = bytes.NewBufferString(*f.literal)
	} else {
		source_file, err := os.Open(*f.source)
		if err != nil {
			return false, fmt.Errorf("opening source %s: %s", *f.source, err)
		}
		defer source_file.Close()
		src = source_file
	}

	prevHash, _, _, err := tr.Do([]string{"sha256sum", f.dest})
	if err != nil {
		return false, fmt.Errorf("error checking file pre content: %s", err)
	}
	_, _, _, err = tr.Do([]string{"rm", "-f", f.dest})
	if err != nil {
		return false, fmt.Errorf("clearing previous file: %s", err)
	}
	_, _, res, err := tr.DoReader([]string{"tee", f.dest}, src)
	if err != nil {
		return false, fmt.Errorf("writing: %s", err)
	}
	if res != 0 {
		return false, fmt.Errorf("non-zero writing: %d", res)
	}

	postHash, _, _, err := tr.Do([]string{"sha256sum", f.dest})
	if err != nil {
		return false, fmt.Errorf("error checking file post content: %s", err)
	}
	if (prevHash == nil && postHash != nil) ||
		(prevHash != nil && postHash != nil && !bytes.Equal(prevHash, postHash)) {
		changed = true
	}
	return changed, nil
}

func (f *File) setMode(tr transport.Transport) (bool, error) {
	if f.mode != nil {
		prevMode, _, _, err := tr.Do([]string{"stat", "-c", "%a", f.dest})
		if err != nil {
			return false, fmt.Errorf("checking file mode: %s", err)
		}
		log.Debugf("prev mode: %s", string(prevMode))
		prevModeVal, err := strconv.ParseInt(strings.TrimSpace(string(prevMode)), 8, 16)
		if err != nil {
			log.Warningf("error parsing previous mode: %s", err)
		}
		if prevModeVal == int64(*f.mode) {
			return false, nil
		}
		_, stderr, res, err := tr.Do([]string{"chmod", fmt.Sprintf("%04o", *f.mode), f.dest})
		if err != nil {
			return false, fmt.Errorf("setting mode %04o on %s: %s", *f.mode, f.dest, err)
		}
		if res != 0 {
			log.Debugf("non-zero setting mod %04o on %s: %d", *f.mode, f.dest, res)
			log.Debugf("stderr: %s", string(stderr))
			return false, fmt.Errorf("non-zero setting mode %04o on %s: %d", *f.mode, f.dest, res)
		}
		return true, nil
	}
	return false, nil
}

func (f *File) setUid(tr transport.Transport) (bool, error) {
	if f.uid != nil {
		prevUid, _, _, err := tr.Do([]string{"stat", "-c", "%u", f.dest})
		if err != nil {
			return false, fmt.Errorf("checking file owner: %s", err)
		}
		prevUidVal, err := strconv.Atoi(strings.TrimSpace(string(prevUid)))
		if err != nil {
			log.Warningf("failed parsing previous UID: %s", err)
		}
		log.Debugf("previous UID: %d", prevUidVal)
		if prevUidVal == int(*f.uid) {
			return false, nil
		}
		_, stderr, res, err := tr.Do([]string{"chown", strconv.Itoa(int(*f.uid)), f.dest})
		if err != nil {
			return false, fmt.Errorf("setting uid %d on %s: %s", *f.uid, f.dest, err)
		}
		if res != 0 {
			log.Debugf("non-zero setting uid %d on %s: %d", *f.uid, f.dest, res)
			log.Debugf("stderr: %s", string(stderr))
			return false, fmt.Errorf("non-zero setting uid %s on %s: %d", *f.uid, f.dest, res)
		}
		return true, nil
	}
	return false, nil
}

func (f *File) setGid(tr transport.Transport) (bool, error) {
	if f.gid != nil {
		prevGid, _, _, err := tr.Do([]string{"stat", "-c", "%g", f.dest})
		if err != nil {
			return false, fmt.Errorf("checking file owner: %s", err)
		}
		prevGidVal, err := strconv.Atoi(strings.TrimSpace(string(prevGid)))
		if err != nil {
			log.Warningf("error parsing previous GID: %s", err)
		}
		if prevGidVal == int(*f.gid) {
			return false, nil
		}
		_, stderr, res, err := tr.Do([]string{"chown", ":" + strconv.Itoa(int(*f.gid)), f.dest})
		if err != nil {
			return false, fmt.Errorf("setting gid %d on %s: %s", *f.gid, f.dest, err)
		}
		if res != 0 {
			log.Debugf("non-zero setting gid %d on %s: %d", *f.gid, f.dest, res)
			log.Debugf("stderr: %s", string(stderr))
			return false, fmt.Errorf("non-zero setting gid %s on %s: %d", *f.gid, f.dest, res)
		}
		return true, nil
	}
	return false, nil
}

func (f *File) Execute(target *types.Target, tr transport.Transport) (bool, error) {
	changed, err := f.writeFile(tr)
	if err != nil {
		return false, err
	}
	log.Debugf("file %s on %s content change => %t", f.dest, target.Name, changed)

	modeChanged, err := f.setMode(tr)
	if err != nil {
		return false, err
	}
	log.Debugf("file %s on %s mode change => %t", f.dest, target.Name, modeChanged)

	uidChanged, err := f.setUid(tr)
	if err != nil {
		return false, err
	}
	log.Debugf("file %s on %s UID change => %t", f.dest, target.Name, uidChanged)

	gidChanged, err := f.setGid(tr)
	if err != nil {
		return false, err
	}
	log.Debugf("file %s on %s GID change => %t", f.dest, target.Name, gidChanged)

	return changed || modeChanged || uidChanged || gidChanged, nil
}

func (*File) Name() string { return "file" }
func (*File) Always() bool { return false }

var _ Module = (*File)(nil)
