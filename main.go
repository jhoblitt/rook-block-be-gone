package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"golang.org/x/crypto/ssh"
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

// Attempt to run command and return error
func run_cmd_raw(client *ssh.Client, cmd string) (string, string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("Failed to create session: %s", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("Failed to run: %s\nstdout:\n%s\nstderr:\n%s\nerror %s", cmd, stdout.String(), stderr.String(), err.Error())
	}

	return stdout.String(), stderr.String(), nil
}

// run command; errors are fatal
func run_cmd(client *ssh.Client, cmd string) {
	fmt.Printf("%s -- running cmd: %s\n", client.RemoteAddr(), cmd)
	stdout, stderr, err := run_cmd_raw(client, cmd)
	if err != nil {
		log.Fatal(err)
	}
	if stdout != "" {
		fmt.Printf("stdout:\n%s", stdout)
	}
	if stderr != "" {
		fmt.Printf("stderr:\n%s", stderr)
	}
}

// run command; ignoring errors
func run_cmd_no_worries(client *ssh.Client, cmd string) {
	fmt.Printf("%s -- running cmd: %s\n", client.RemoteAddr(), cmd)
	stdout, stderr, _ := run_cmd_raw(client, cmd)
	if stdout != "" {
		fmt.Printf("stdout:\n%s", stdout)
	}
	if stderr != "" {
		fmt.Printf("stderr:\n%s", stderr)
	}
}

func main() {
	default_ssh_priv_key := fmt.Sprintf("%s/.ssh/id_rsa", os.Getenv("HOME"))
	ssh_priv_key := flag.String("ssh-private-key", default_ssh_priv_key, "path to ssh private key")
	ssh_user := flag.String("ssh-user", os.Getenv("USERNAME"), "ssh username")
	flag.Parse()

	yfile, err := ioutil.ReadFile("rook-ceph-cluster-values.yaml")
	if err != nil {
		log.Fatal(err)
	}

	var data Doc
	err2 := yaml.Unmarshal(yfile, &data)
	if err2 != nil {
		log.Fatal(err2)
	}

	key, err := os.ReadFile(*ssh_priv_key)
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: *ssh_user,
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
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
		client, err := ssh.Dial("tcp", v.Name+":22", config)
		if err != nil {
			log.Fatalf("unable to connect: %v", err)
		}
		defer client.Close()

		run_cmd(client, "sudo /bin/rm -rf /var/lib/rook")
		run_cmd(client, "ls /dev/mapper/ceph-* | xargs -I%% -- echo sudo /sbin/dmsetup remove %% | sh")
		run_cmd(client, "sudo /bin/rm -rf /dev/ceph-*")

		dev_prefix := regexp.MustCompile("^/dev/")

		for _, v := range v.Devices {
			// check if device name includes a /dev/ prefix
			var dev string
			if dev_prefix.MatchString(v.Name) {
				dev = v.Name
			} else {
				dev = fmt.Sprintf("/dev/%s", v.Name)
			}

			run_cmd(client, fmt.Sprintf("sudo /sbin/sgdisk --zap-all \"%s\"", dev))
			run_cmd(client, fmt.Sprintf("sudo /bin/dd if=\"/dev/zero\" of=\"%s\" bs=1M count=100 oflag=direct,dsync", dev))
			run_cmd(client, fmt.Sprintf("sudo /sbin/blockdev --rereadpt \"%s\"", dev))
		}

		run_cmd(client, "/bin/lsblk")
		run_cmd_no_worries(client, "sudo /sbin/reboot")
	}
}
