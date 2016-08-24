package glusterfs

import (
    "fmt"
    "github.com/Sirupsen/logrus"
    "github.com/rancher/convoy/util"
    "math/rand"
    "path/filepath"
    "strconv"
    "strings"
    "sync"
    "github.com/rancher/convoy/cephfs/cephfslib"
    . "github.com/rancher/convoy/convoydriver"
)

const (
    DRIVER_NAME        = "cephfs"
    DRIVER_CONFIG_FILE = "cephfs.cfg"

    VOLUME_CFG_PREFIX = "volume_"
    DRIVER_CFG_PREFIX = DRIVER_NAME + "_"
    CFG_POSTFIX       = ".json"

    SNAPSHOT_PATH = "snapshots"

    MOUNTS_DIR = "mounts"

    CEPHFS_SERVERS             = "cephfs.servers"
)

var (
    log = logrus.WithFields(logrus.Fields{"pkg": "cephfs"})
)

type Driver struct {
    mutex    *sync.RWMutex
    Device
}

func init() {
    Register(DRIVER_NAME, Init)
}

func (d *Driver) Name() string {
    return DRIVER_NAME
}

type Device struct {
    Root              string
    Servers           string
}

func (dev *Device) ConfigFile() (string, error) {
    if dev.Root == "" {
        return "", fmt.Errorf("BUG: Invalid empty device config path")
    }
    return filepath.Join(dev.Root, DRIVER_CONFIG_FILE), nil
}

type Volume struct {
    Name         string
    Path         string
    MountPoint   string
    CreatedTime  string
    configPath string
}

type CephfsFSVolume struct {
    MountPoint string
    Servers    []string
    configPath string
}

func (gv *CephfsFSVolume) GetDevice() (string, error) {
    l := len(gv.Servers)
    if gv.Servers == nil || len(gv.Servers) == 0 {
        return "", fmt.Errorf("No server IP provided for glusterfs")
    }
    ip := gv.Servers[rand.Intn(l)]
    return ip + ":/"+gv.MountPoint, nil
}

func (gv *CephfsFSVolume) GetMountOpts() []string {
    return []string{"-t", "ceph"}
}

func (gv *CephfsFSVolume) GenerateDefaultMountPoint() string {
    return filepath.Join(gv.configPath, MOUNTS_DIR)
}

func (v *Volume) ConfigFile() (string, error) {
    if v.Name == "" {
        return "", fmt.Errorf("BUG: Invalid empty volume name")
    }
    if v.configPath == "" {
        return "", fmt.Errorf("BUG: Invalid empty volume config path")
    }
    return filepath.Join(v.configPath, DRIVER_CFG_PREFIX+VOLUME_CFG_PREFIX+v.Name+CFG_POSTFIX), nil
}

func (device *Device) listVolumeNames() ([]string, error) {
    return util.ListConfigIDs(device.Root, DRIVER_CFG_PREFIX+VOLUME_CFG_PREFIX, CFG_POSTFIX)
}

func Init(root string, config map[string]string) (ConvoyDriver, error) {
    dev := &Device{
        Root: root,
    }
    exists, err := util.ObjectExists(dev)
    if err != nil {
        return nil, err
    }
    if cephfslib.InitCommomLib() == false {
        return nil, fmt.Errorf("InitCommomLib err ")
    }
    if exists {
        if err := util.ObjectLoad(dev); err != nil {
            return nil, err
        }
    } else {
        if err := util.MkdirIfNotExists(root); err != nil {
            return nil, err
        }

        serverList := config[CEPHFS_SERVERS]
        if serverList == "" {
            return nil, fmt.Errorf("Missing required parameter: %v", CEPHFS_SERVERS)
        }

        servers := strings.Split(serverList, ",")
        for _, server := range servers {
            if !util.ValidNetworkAddr(server) {
                return nil, fmt.Errorf("Invalid or unsolvable address: %v", server)
            }
        }
        dev = &Device{
            Root:              root,
            Servers:           serverList,
        }
    }
    d := &Driver{
        mutex:    &sync.RWMutex{},
        Device:   *dev,
    }
    return d, nil
}

func (d *Driver) Info() (map[string]string, error) {
    return map[string]string{
        "Root":              d.Root,
        "CephFSServers":  fmt.Sprintf("%v", d.Servers),
    }, nil
}

func (d *Driver) VolumeOps() (VolumeOperations, error) {
    return d, nil
}

func (d *Driver) blankVolume(name string) *Volume {
    return &Volume{
        configPath: d.Root,
        Name:       name,
    }
}

func (d *Driver) CreateVolume(req Request) error {
    d.mutex.Lock()
    defer d.mutex.Unlock()

    id := req.Name
    volume := d.blankVolume(id)
    exists, err := util.ObjectExists(volume)
    if err != nil {
        return err
    }
    if exists {
        return fmt.Errorf("volume %v already exists", id)
    }

    volumePath := filepath.Join(d.Device.Root, id)

    if cephfslib.IsMounted(volumePath) {
        log.Debugf("Found existing volume named %v, reuse it", volumePath)
    }
    volume.Name = id
    volume.Path = volumePath
    volume.CreatedTime = util.Now()

    return util.ObjectSave(volume)
}

func (d *Driver) DeleteVolume(req Request) error {
    d.mutex.Lock()
    defer d.mutex.Unlock()

    id := req.Name
    opts := req.Options

    volume := d.blankVolume(id)
    if err := util.ObjectLoad(volume); err != nil {
        return err
    }

    if volume.MountPoint != "" {
        return fmt.Errorf("Cannot delete volume %v. It is still mounted", id)
    }
    referenceOnly, _ := strconv.ParseBool(opts[OPT_REFERENCE_ONLY])
    if !referenceOnly {
        log.Debugf("Cleaning up volume %v", id)
        if cephfslib.VolumeMountPointDirectoryRemove(volume.Path) == false {
            return fmt.Errorf("VolumeMountPointDirectoryRemove Cannot delete volume %v",volume.Path)
        }
        if err := cephfslib.UnmountVolume(volume.Path);err != nil{
            return err
        }
    }
    return util.ObjectDelete(volume)
}

func (d *Driver) MountVolume(req Request) (string, error) {
    d.mutex.Lock()
    defer d.mutex.Unlock()
    id := req.Name

    volume := d.blankVolume(id)
    if err := util.ObjectLoad(volume); err != nil {
        return "", err
    }

    if volume.MountPoint == "" {
        volume.MountPoint = volume.Path
    }
    log.Debugf("call MountVolume %s %s %s %s",d.Device.Servers,volume.Name,volume.MountPoint,volume.Path)
    if cephfslib.MountCephFs(d.Device.Servers,volume.Name,volume.MountPoint,"/etc/ceph/ceph.client.admin.keyring") == false {
        return "",fmt.Errorf("MountCephFs error")
    }

    if err := util.ObjectSave(volume); err != nil {
        return "", err
    }
    return volume.MountPoint, nil
}

func (d *Driver) UmountVolume(req Request) error {
    d.mutex.Lock()
    defer d.mutex.Unlock()

    id := req.Name

    volume := d.blankVolume(id)
    if err := util.ObjectLoad(volume); err != nil {
        return err
    }

    if volume.MountPoint != "" {
        volume.MountPoint = ""
    }
    return util.ObjectSave(volume)
}

func (d *Driver) ListVolume(opts map[string]string) (map[string]map[string]string, error) {
    d.mutex.RLock()
    defer d.mutex.RUnlock()

    volumeIDs, err := d.listVolumeNames()
    if err != nil {
        return nil, err
    }
    result := map[string]map[string]string{}
    for _, id := range volumeIDs {
        result[id], err = d.GetVolumeInfo(id)
        if err != nil {
            return nil, err
        }
    }
    return result, nil
}

func (d *Driver) GetVolumeInfo(id string) (map[string]string, error) {
    d.mutex.RLock()
    defer d.mutex.RUnlock()

    volume := d.blankVolume(id)
    if err := util.ObjectLoad(volume); err != nil {
        return nil, err
    }
    return map[string]string{
        OPT_VOLUME_NAME:         volume.Name,
        "Path":                  volume.Path,
        OPT_MOUNT_POINT:         volume.MountPoint,
        OPT_VOLUME_CREATED_TIME: volume.CreatedTime,
        "CephFSServers":      fmt.Sprintf("%v", d.Device.Servers),
    }, nil
}

func (d *Driver) MountPoint(req Request) (string, error) {
    d.mutex.RLock()
    defer d.mutex.RUnlock()

    id := req.Name

    volume := d.blankVolume(id)
    if err := util.ObjectLoad(volume); err != nil {
        return "", err
    }
    return volume.MountPoint, nil
}

func (d *Driver) SnapshotOps() (SnapshotOperations, error) {
    return nil, fmt.Errorf("Doesn't support snapshot operations")
}

func (d *Driver) BackupOps() (BackupOperations, error) {
    return nil, fmt.Errorf("Doesn't support backup operations")
}
