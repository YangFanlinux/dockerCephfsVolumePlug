package main

import (
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "github.com/docker/go-plugins-helpers/volume"
    "github.com/YangFanlinux/dockerCephfsVolumePlug/cephfslib"
)


const cephfsID = "_cephfs"
var (
    defaultDir  = filepath.Join(volume.DefaultDockerRootDirectory, cephfsID)
    servers = flag.String("servers", "", "List of CephMonitor IP:port")
    root        = flag.String("root", defaultDir, "CephFS volumes root directory")
)

func main() {
    var Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage for docker data volume plugin: %s [options]\n", os.Args[0])
        flag.PrintDefaults()
    }

    if cephfslib.InitCommomLib() == false {
         os.Exit(1)
    }
    defer cephfslib.ReleaseCommonLib()
    cephfslib.DebugLog.Printf("Start dockerCephfsVolumePlug...")
    flag.Parse()
    if len(*servers) != 0 {
        d := NewCephfsDriver(*root, *servers)
        h := volume.NewHandler(d)
        fmt.Println(h.ServeUnix("root", "cephfs"))
    }else if flag.NArg() > 0 {
        // Is k8s Interface?
        switch flag.Arg(0) {
        case "init":
            FlexInit()
        case "attach":
            FlexAttach(flag.Arg(1))
        case "detach":
            FlexDetach()
        case "mount":
            FlexMount(flag.Arg(1),flag.Arg(2),flag.Arg(3))
        case "unmount":
            FlexUnmount(flag.Arg(1))
        default:
            Usage()
            os.Exit(1)
        }
    }else{
        Usage()
        os.Exit(1)
    }
}
