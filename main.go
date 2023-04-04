package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

type Doc struct {
	CephClusterSpec struct {
		Storage struct {
			Nodes []struct {
				Name    string `yaml:"name"`
				Devices []struct {
					Name string `yaml:"name"`
				}
			} `yaml:"nodes"`
		} `yaml:"storage"`
	} `yaml:"cephClusterSpec"`
}

func main() {
	yfile, err := ioutil.ReadFile("rook-ceph-cluster-values.yaml")
	if err != nil {
		log.Fatal(err)
	}

	var data Doc
	err2 := yaml.Unmarshal(yfile, &data)
	if err2 != nil {
		log.Fatal(err2)
	}

	/*
		for _, v := range data.CephClusterSpec.Storage.Nodes {
			fmt.Printf("%s\n", v.Name)
			for _, v := range v.Devices {
				fmt.Printf("%s\n", v.Name)
			}
		}
	*/

	for _, v := range data.CephClusterSpec.Storage.Nodes {
		host := "rke@" + v.Name

		fmt.Printf("ssh %s sudo /bin/rm -rf /var/lib/rook\n", host)
		fmt.Printf("ssh %s 'ls /dev/mapper/ceph-* | xargs -I%% -- echo sudo /sbin/dmsetup remove %% | sh'\n", host)
		fmt.Printf("ssh %s sudo /bin/rm -rf /dev/ceph-*\n", host)

		for _, v := range v.Devices {
			dev := v.Name

			fmt.Printf("ssh %s sudo /sbin/sgdisk --zap-all \"/dev/%s\"\n", host, dev)
			fmt.Printf("ssh %s sudo /bin/dd if=\"/dev/zero\" of=\"/dev/%s\" bs=1M count=100 oflag=direct,dsync\n", host, dev)
			fmt.Printf("ssh %s sudo /sbin/blockdev --rereadpt \"/dev/%s\"\n", host, dev)
		}

		fmt.Printf("ssh %s /bin/lsblk\n", host)
		fmt.Printf("ssh %s sudo /sbin/reboot\n", host)
	}
}
