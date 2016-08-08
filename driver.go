package main

import (
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "github.com/docker/go-plugins-helpers/volume"
)

type volumeName struct {
    name        string
    connections int
}

type cephfsDriver struct {
    root       string
    servers    string
    volumes    map[string]*volumeName
    m          *sync.Mutex
}



func NewCephfsDriver(root string, servers string) cephfsDriver {
    d := cephfsDriver{
        root:    root,
        servers: servers,
        volumes: map[string]*volumeName{},
        m:       &sync.Mutex{},
    }
    return d
}

/*func (d *cephfsDriver) MountRootForCephfs() bool {
    var retError error
    d.m.Lock()
    defer d.m.Unlock()

    return MountCephFs(d.servers,d.root,"/etc/ceph/ceph.client.admin.keyring")

    fi, err := os.Lstat(d.root)
    if os.IsNotExist(err) {
        if err := os.MkdirAll(d.root, 0755); err != nil {
            return err
        }
    } else if err != nil {
        return err
    }

    if fi != nil && !fi.IsDir() {
        return errors.New("mountPoint is not a dir")
    }
    if d.isMounted(d.root) {
        return nil
    }
    for _, server := range d.servers {
        debugLog.Printf(" mountVolume ip %s",server)
        cmd := fmt.Sprintf("mount -t ceph %s:/ %s -o name=admin,secret=`ceph-authtool -p /etc/ceph/ceph.client.admin.keyring`", server,d.root)
        debugLog.Println(string(cmd))
        if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
            debugLog.Println(string(out))
            retError = err
        }else {
            return nil
        }
    }
    return retError
}*/

func (d cephfsDriver) Create(r volume.Request) volume.Response {
    debugLog.Printf("Creating volume %s\n", r.Name)
    d.m.Lock()
    defer d.m.Unlock()
    m := d.mountpoint(r.Name)
    if _, ok := d.volumes[m]; ok {
        return volume.Response{}
    }
    return volume.Response{}
}
//just remove key-value
func (d cephfsDriver) Remove(r volume.Request) volume.Response {
    debugLog.Printf("Removing volume %s\n", r.Name)
    d.m.Lock()
    defer d.m.Unlock()
    m := d.mountpoint(r.Name)

    if s, ok := d.volumes[m]; ok {
        if s.connections <= 1 {
            delete(d.volumes, m)
        }
    }
    return volume.Response{}
}

func (d cephfsDriver) Path(r volume.Request) volume.Response {
    return volume.Response{Mountpoint: d.mountpoint(r.Name)}
}

func (d cephfsDriver) Mount(r volume.Request) volume.Response {
    d.m.Lock()
    defer d.m.Unlock()

    m := d.mountpoint(r.Name)
    //sudo ./docker-volume-cephfs -servers 127.0.0.1:6789 -root /mnt/mycephfs/
    //sudo docker run --volume-driver dockerCephfsVolumePlug --volume dockerVolume4:/data ubuntu touch /data/zzz
    // --volume dockerVolume4:/data    -root /mnt
    //r.Name  = dockerVolume4 m = /mnt/datastore
    debugLog.Printf("Mounting volume %s on %s\n", r.Name, m)

    s, ok := d.volumes[m]
    if ok && s.connections > 0 {
        s.connections++
        return volume.Response{Mountpoint: m}
    }

    fi, err := os.Lstat(m)

    if os.IsNotExist(err) {
        if err := os.MkdirAll(m, 0755); err != nil {
            return volume.Response{Err: err.Error()}
        }
    } else if err != nil {
        return volume.Response{Err: err.Error()}
    }

    if fi != nil && !fi.IsDir() {
        return volume.Response{Err: fmt.Sprintf("%v already exist and it's not a directory", m)}
    }
    if MountCephFs(d.servers,r.Name,m,"/etc/ceph/ceph.client.admin.keyring") == false {
        return volume.Response{Err: fmt.Sprintf("%v mount cephfs err.Maybe dir() isn't in /", r.Name)}
    }
    d.volumes[m] = &volumeName{name: r.Name, connections: 1}

    return volume.Response{Mountpoint: m}
}

func (d cephfsDriver) Unmount(r volume.Request) volume.Response {
    d.m.Lock()
    defer d.m.Unlock()
    m := d.mountpoint(r.Name)
    debugLog.Printf("Unmounting volume %s from %s\n", r.Name, m)

    if s, ok := d.volumes[m]; ok {
        if s.connections == 1 {
            if err := unmountVolume(m); err != nil {
                return volume.Response{Err: err.Error()}
            }
        }
        s.connections--
    } else {
        return volume.Response{Err: fmt.Sprintf("Unable to find volume mounted on %s", m)}
    }

    return volume.Response{}
}

func (d cephfsDriver) Get(r volume.Request) volume.Response {
    d.m.Lock()
    defer d.m.Unlock()
    m := d.mountpoint(r.Name)
    if s, ok := d.volumes[m]; ok {
        return volume.Response{Volume: &volume.Volume{Name: s.name, Mountpoint: d.mountpoint(s.name)}}
    }

    return volume.Response{Err: fmt.Sprintf("Unable to find volume mounted on %s", m)}
}

func (d cephfsDriver) List(r volume.Request) volume.Response {
    d.m.Lock()
    defer d.m.Unlock()
    var vols []*volume.Volume
    for _, v := range d.volumes {
        vols = append(vols, &volume.Volume{Name: v.name, Mountpoint: d.mountpoint(v.name)})
    }
    return volume.Response{Volumes: vols}
}

func (d *cephfsDriver) mountpoint(name string) string {
    return filepath.Join(d.root, name)
}

/*
func (d *cephfsDriver) unmountVolume(mountPath string) error {
    cmd := fmt.Sprintf("umount %s", mountPath)
    if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
        log.Println(string(out))
        return err
    }
    return nil
}

func (d *cephfsDriver) isMounted(mountPath string) bool {
    cmd := fmt.Sprintf("findmnt -n %s", mountPath)
    mountPath = strings.TrimRight(mountPath,"/")
    if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err == nil {
        lines := strings.Split(string(out),"\n")
        for _, line := range lines {
            if strings.Contains(line,mountPath) {
                debugLog.Printf("isMount %s",line)
                if strings.Contains(line,"ceph") {
                    debugLog.Printf("%s had already mounted(%s)",mountPath,line)
                    return true
                }else{
                    d.unmountVolume(mountPath)
                }
            }
        }
    }
    return false
}*/
