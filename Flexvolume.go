package main
import (
    "encoding/json"
    "os"
    "fmt"
)
/*
type cephConnectInfo struct{
    mountPath string
    serverList []string
    keyFilePath string
}*/
/*
func B2S(buf []byte) string {
    return *(*string)(unsafe.Pointer(&buf))
}

func S2B(s *string) []byte {
    return *(*[]byte)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(s))))
}
*/
func FlexInit() {
    os.Stdout.WriteString("{\"status\": \"Success\"}")
    return
}

func FlexAttach(jsonValue string) {
    os.Stdout.WriteString("{\"status\": \"Success\"}")
    return
}


func FlexDetach() {
    os.Stdout.WriteString("{\"status\": \"Success\"}")
    return
}

func FlexMount(hostMountPath string,flexAttachPath string,jsonValue string) {
    var obj interface{}
    json.Unmarshal([]byte(jsonValue),&obj)
    m := obj.(map[string]interface{})
    monitors := m["monitors"].(string)
    pathInCeph := m["path"].(string)
    secretFile := m["secretFile"].(string)
    debugLog.Printf(" arg1 = %s,arg2 = %s",hostMountPath,flexAttachPath)

    if len(m["monitors"].(string)) != 0 {
        if MountCephFs(monitors,pathInCeph,hostMountPath,secretFile) {
            os.Stdout.WriteString("{\"status\": \"Success\"}")
            return
        }else{
            os.Stdout.WriteString("{\"status\": \"Failure\",\"message\": \"can't mount cephfs\"}")
            return
        }
    }else{
        os.Stdout.WriteString("{\"status\": \"Failure\",\"message\": \"can't get monitor ip\"}")
        return
    }
}

func FlexUnmount(hostMountPath string) {
    if isMounted(hostMountPath) {
        if err :=unmountVolume(hostMountPath);err != nil{
            os.Stdout.WriteString(fmt.Sprintf("{\"status\": \"Failure\",\"message\": \"can't unmount %s.err = %s\"}",hostMountPath,err.Error()))
            return
        }
    }
    os.Stdout.WriteString("{\"status\": \"Success\"}")
    return
}
