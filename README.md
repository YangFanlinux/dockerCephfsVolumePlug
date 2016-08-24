This project is used for docker data plugin to support docker plugin,kubernetes FlexVolume and docker convoy.
Compile:
	For kubernetes or docker plugin:
	1)copy src to $GOPATH
	  eg:cp dockerCephfsVolumePlug/ ./GOPATH/src/github.com/YangFanlinux/dockerCephfsVolumePlug
	2)cd ./GOPATH/src/github.com/YangFanlinux/dockerCephfsVolumePlug and "go build"
	3)You can find the bin named dockerCephfsVolumePlug in ./GOPATH/src/github.com/YangFanlinux/dockerCephfsVolumePlug
	For convoy
	1)copy dockerCephfsVolumePlug/cephfs to GOPATH/src/github.com/rancher/convoy/
	  and copy dockerCephfsVolumePlug/cephfslib to GOPATH/src/github.com/rancher/convoy/cephfs/
	2)"make build" in GOPATH/src/github.com/rancher/convoy/

Usage:

	For kubernetes:
	1)Create yaml file like nginx.yaml
	2)cp dockerCephfsVolumePlug to /usr/libexec/kubernetes/kubelet-plugins/volume/exec/neunn~dockerCephfsVolumePlug/
	3)Create pod "kubectl create -f ./nginx.yaml"
	For docker plugin:
	1)sudo ./dockerCephfsVolumePlug -servers monitorIp:port -root cephfsDir
	  eg: sudo ./dockerCephfsVolumePlug -servers 192.168.0.22,192.168.0.23 -root /mnt/cephfs/
	  sudo docker run --volume-driver cephfs --volume aaa:/mnt/ ubuntu touch /data/testDir
    For convoy:
	1)sudo ./convoy daemon --drivers cephfs --driver-opts cephfs.servers=127.0.0.1
	  sudo ./convoy  create volumename1
      sudo docker run -v volumename1:/mnt/ --volume-driver=convoy ubuntu touch /mnt/bbb
	  sudo ./convoy delete volumename1