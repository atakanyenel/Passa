//Package cloudsolution provides function for docker swarm
package cloudsolution

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"gitlab.lrz.de/ga53lis/PASSA/ymlparser"

	"golang.org/x/crypto/ssh"
)

//DockerSwarm keeps joinToken and managerIP of the system
type DockerSwarm struct {
	joinToken string
	managerIP string
}

//NewSwarmManager returns a dockerswarm manager
func NewSwarmManager(managerIP string) DockerSwarm {

	return DockerSwarm{managerIP: managerIP, joinToken: getWorkerToken(managerIP)}
}

func createNewMachine(machineName string) []byte {
	cmd := exec.Command("sh", "-c", "docker-machine create --driver virtualbox "+machineName)

	stdout, _ := cmd.StdoutPipe()
	cmd.Start()
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}
	cmd.Wait()

	out, _ := cmd.Output()
	return out
}

func getNewMachineIP(machineName string) string {
	newIP, err := exec.Command("sh", "-c", "docker-machine ip "+machineName).Output()

	if err != nil {
		panic(err)
	}
	return strings.Trim(string(newIP[:]), "\n")
}

func getWorkerToken(managerIP string) string {

	keyFile := fmt.Sprintf("%s/.docker/machine/machines/%s/id_rsa", os.Getenv("HOME"), "myvm1")
	key, err := ioutil.ReadFile(keyFile)

	signer, err := ssh.ParsePrivateKey(key)
	config := &ssh.ClientConfig{
		User: "docker",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //FIXME: fix security
	}

	fmt.Println(managerIP)
	client, err := ssh.Dial("tcp", managerIP+":22", config)

	if err != nil {
		log.Fatal("Failed to dial: ", err)
	}
	session, err := client.NewSession()

	if err != nil {
		log.Fatal("Failed to session: ", err)
	}

	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run("docker swarm join-token --quiet worker"); err != nil {
		log.Fatal("Failed to run:" + err.Error())
	}

	return b.String()
}

func addToSwarm(joinToken string, newMachineIP string, managerIP string, machineName string) string {
	keyFile := fmt.Sprintf("%s/.docker/machine/machines/%s/id_rsa", os.Getenv("HOME"), machineName)
	fmt.Println(keyFile)
	key, err := ioutil.ReadFile(keyFile)

	signer, err := ssh.ParsePrivateKey(key)
	config := &ssh.ClientConfig{
		User: "docker",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //FIXME: fix security
	}

	fmt.Println(newMachineIP)
	client, err := ssh.Dial("tcp", newMachineIP+":22", config)

	if err != nil {
		log.Fatal("Failed to dial: ", err)
	}
	session, err := client.NewSession()

	if err != nil {
		log.Fatal("Failed to session: ", err)
	}

	defer session.Close()

	var b bytes.Buffer

	session.Stdout = &b
	swarmCommand := fmt.Sprintf("docker swarm join --token %s %s:2377", strings.Trim(joinToken, "\n"), managerIP)
	fmt.Println(swarmCommand)
	if err := session.Run(swarmCommand); err != nil {
		log.Fatal("Failed to run:" + err.Error())
	}

	return b.String()
}

func scaleContainers(managerIP string, containerName string, scaleNum string) string {
	keyFile := fmt.Sprintf("%s/.docker/machine/machines/%s/id_rsa", os.Getenv("HOME"), "myvm1")
	key, err := ioutil.ReadFile(keyFile)

	signer, err := ssh.ParsePrivateKey(key)
	config := &ssh.ClientConfig{
		User: "docker",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //FIXME: fix security
	}

	fmt.Println(managerIP)
	client, err := ssh.Dial("tcp", managerIP+":22", config)

	if err != nil {
		log.Fatal("Failed to dial: ", err)
	}
	session, err := client.NewSession()

	if err != nil {
		log.Fatal("Failed to session: ", err)
	}

	defer session.Close()

	//var b bytes.Buffer
	//session.Stdout = &b
	stdout, _ := session.StdoutPipe()

	scalingCommand := fmt.Sprintf("docker service scale %s=%s", containerName, scaleNum)

	if err := session.Start(scalingCommand); err != nil {
		log.Fatal("Failed to run:" + err.Error())
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}

	session.Wait()
	return ""
}

func deleteMachine(machineName string) []byte {
	cmd := exec.Command("sh", "-c", "docker-machine rm "+machineName+" -y")

	stdout, _ := cmd.StdoutPipe()
	cmd.Start()
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}
	cmd.Wait()

	out, _ := cmd.Output()
	return out
}

func listMachines() []string {
	cmd := exec.Command("sh", "-c", "docker-machine ls -q")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	machinesList := strings.Split(strings.Trim(string(out[:]), "\n"), "\n")

	return machinesList
}

//ChangeState changes the state of the system
func (ds DockerSwarm) ChangeState(wantedState ymlparser.Service) []string {

	currentState := listMachines()
	scaleInt, err := strconv.Atoi(wantedState.Scale)
	if err != nil {
		panic(err)
	}
	difference := len(currentState) - scaleInt
	fmt.Println(difference)
	if difference == 0 { //keep the state as is
		return currentState
	} else if difference > 0 { //lets delete some machines
		var wg sync.WaitGroup
		wg.Add(difference)
		for i := 0; i < difference; i++ {
			lastCompName := currentState[len(currentState)-1-i]
			go func() {
				defer wg.Done()
				deleteMachine(lastCompName)
				removeFromSwarm("192.168.99.100", lastCompName)
			}()

		}
		wg.Wait()
	} else { //difference <0 , lets add some machines
		var wg sync.WaitGroup
		wg.Add(-difference)
		for i := 0; i < -1*difference; i++ {
			newMachineName := fmt.Sprintf("myvm%v", len(currentState)+i+1)
			fmt.Println(newMachineName)
			go func() {
				defer wg.Done()
				createNewMachine(newMachineName)
				newIP := getNewMachineIP(newMachineName)
				joinToken := getWorkerToken("192.168.99.100")
				addToSwarm(joinToken, newIP, "192.168.99.100", newMachineName)

			}()

		}
		wg.Wait()
		scaleContainers(ds.managerIP, wantedState.Name, wantedState.Scale)

	}

	return listMachines()
}

func removeFromSwarm(managerIP string, machineName string) string {
	keyFile := fmt.Sprintf("%s/.docker/machine/machines/%s/id_rsa", os.Getenv("HOME"), "myvm1")
	key, err := ioutil.ReadFile(keyFile)

	signer, err := ssh.ParsePrivateKey(key)
	config := &ssh.ClientConfig{
		User: "docker",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //FIXME: fix security
	}

	fmt.Println(managerIP)
	client, err := ssh.Dial("tcp", managerIP+":22", config)

	if err != nil {
		log.Fatal("Failed to dial: ", err)
	}
	session, err := client.NewSession()

	if err != nil {
		log.Fatal("Failed to session: ", err)
	}

	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run("docker node rm -f " + machineName); err != nil {
		log.Fatal("Failed to run:" + err.Error())
	}

	return b.String()
}
