package cephfslib

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
    "log"
)

var DebugLog *log.Logger
var logFile *os.File

func InitCommomLib() bool {
    fileName := "docker-volume-cephfs.log"
    var err error
    logFile,err = os.OpenFile(fileName,os.O_RDWR | os.O_APPEND | os.O_CREATE, 0666)
    if err != nil {
        log.Fatalln("open docker-volume-cephfs.log file error !")
        return false
    }
    DebugLog = log.New(logFile,"[Debug]",log.LstdFlags)
    if DebugLog == nil {

        log.Fatalln("New DebugLog error!")
        return false
    }
    return true
}

func ReleaseCommonLib() {
    logFile.Close()
}

func UnmountVolume(mountPath string) error {
    cmd := fmt.Sprintf("umount %s", mountPath)
    if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
        DebugLog.Println(string(out))
        return err
    }
    return nil
}

func IsMounted(mountPath string) bool {
    cmd := fmt.Sprintf("findmnt -n %s", mountPath)
    mountPath = strings.TrimRight(mountPath,"/")
    if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err == nil {
        lines := strings.Split(string(out),"\n")
        for _, line := range lines {
            if strings.Contains(line,mountPath) {
                DebugLog.Printf("isMount %s",line)
                if strings.Contains(line,"ceph") {  //The judgement isn't strict.
                    DebugLog.Printf("%s had already mounted(%s)",mountPath,line)
                    return true
                }else{
                    UnmountVolume(mountPath)
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
            DebugLog.Printf("MkdirAll err(%s)",err)
            return false
        }
    } else if err != nil {
        DebugLog.Printf("Lstat err(%s)",err)
        return false
    }
    if fi != nil && !fi.IsDir() {
        return false
    }
    if IsMounted(hostMountPoint) {
        return true
    }
    if strings.HasPrefix(pathInCeph,"/") == false {
        //the dir must be absolute path like "/xxx" in cephfs "/"
        pathInCeph = "/" + pathInCeph
    }
    DebugLog.Printf(" mountVolume ip %s",servers)
    //mount -t ceph  192.168.220.11,127.0.0.1:6789:/aaa/ /mnt/subDir/ -o name=admin,secret=`ceph-authtool -p /etc/ceph/ceph.client.admin.keyring`
    cmd := fmt.Sprintf("mount -t ceph %s:%s %s -o name=admin,secret=`ceph-authtool -p %s`", servers,pathInCeph,hostMountPoint,keyFilePath)
    DebugLog.Println(string(cmd))
    if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
        DebugLog.Println(string(out))
        return false
    }
    DebugLog.Printf("MountCephFs Sucessful.")
    return true
}

func VolumeMountPointDirectoryRemove(dirName string) bool {
    if IsMounted(dirName) == false {
        panic("BUG: VolumeMountPointDirectoryRemove was called before volume mounted")
    }
    cmd := fmt.Sprintf("rm -rf %s/*",dirName)
    DebugLog.Println(string(cmd))
    if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
        DebugLog.Println(string(out))
        return false
    }
    return true
}

