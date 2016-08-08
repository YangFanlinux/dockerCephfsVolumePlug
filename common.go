package main

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
)

func unmountVolume(mountPath string) error {
    cmd := fmt.Sprintf("umount %s", mountPath)
    if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
        debugLog.Println(string(out))
        return err
    }
    return nil
}

func isMounted(mountPath string) bool {
    cmd := fmt.Sprintf("findmnt -n %s", mountPath)
    mountPath = strings.TrimRight(mountPath,"/")
    if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err == nil {
        lines := strings.Split(string(out),"\n")
        for _, line := range lines {
            if strings.Contains(line,mountPath) {
                debugLog.Printf("isMount %s",line)
                if strings.Contains(line,"ceph") {  //The judgement isn't strict.
                    debugLog.Printf("%s had already mounted(%s)",mountPath,line)
                    return true
                }else{
                    unmountVolume(mountPath)
                }
            }
        }
    }
    return false
}

func MountCephFs (servers string,pathInCeph string,hostMountPoint string,keyFilePath string) bool {
    fi, err := os.Lstat(hostMountPoint)
    if os.IsNotExist(err) {
        if err := os.MkdirAll(hostMountPoint, 0755); err != nil {
            debugLog.Printf("MkdirAll err(%s)",err)
            return false
        }
    } else if err != nil {
        debugLog.Printf("Lstat err(%s)",err)
        return false
    }
    if fi != nil && !fi.IsDir() {
        return false
    }
    if isMounted(hostMountPoint) {
        return true
    }
    if strings.HasPrefix(pathInCeph,"/") == false {
        //the dir must be absolute path like "/xxx" in cephfs "/"
        pathInCeph = "/" + pathInCeph
    }
    debugLog.Printf(" mountVolume ip %s",servers)
    //mount -t ceph  192.168.220.11,127.0.0.1:6789:/aaa/ /mnt/subDir/ -o name=admin,secret=`ceph-authtool -p /etc/ceph/ceph.client.admin.keyring`
    cmd := fmt.Sprintf("mount -t ceph %s:%s %s -o name=admin,secret=`ceph-authtool -p %s`", servers,pathInCeph,hostMountPoint,keyFilePath)
    debugLog.Println(string(cmd))
    if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
        debugLog.Println(string(out))
        return false
    }
    debugLog.Printf("MountCephFs Sucessful.")
    return true
}
