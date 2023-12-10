package main

import (
	"bufio"
	"encoding/json"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	CMPATH     = "data/cm.json"
	CLIENTPATH = "data/client.json"
	NUM_REQS   = 10
)

// Color coded logs
var logsystem = color.New(color.FgCyan).Add(color.BgBlack)
var logerror = color.New(color.FgHiRed).Add(color.BgBlack)
var logwarning = color.New(color.FgYellow).Add(color.BgBlack)
var logincoming = color.New(color.FgHiMagenta).Add(color.BgBlack)
var logoutgoing = color.New(color.FgHiYellow).Add(color.BgBlack)

func main() {
	ipAddress := GetOutboundIP().String()
	port, err := GetFreePort()
	if err != nil {
		logerror.Println("Error assigning port number: ", err)
		return
	}
	portStr := strconv.Itoa(port)
	ipPlusPort := ipAddress + ":" + portStr
	logsystem.Println("Node running on IP Address: ", ipPlusPort)

	// Specify type of Node: {Client, Central Manager}
	reader := bufio.NewReader(os.Stdin)
	logsystem.Println("Enter Node type ('1': CM, '2': Client, 'restartCM', 'restartBackup')")
	nodeType, err := reader.ReadString('\n')
	if err != nil {
		logerror.Println("Error reading input: ", err)
		return
	}
	nodeType = strings.TrimRight(nodeType, "\n")

	switch nodeType {
	case "1":
		StartCM(ipPlusPort)
	case "2":
		StartClient(ipPlusPort)
	case "restartCM":
		RestartPrimaryCM()
	case "restartBackup":
		RestartBackupCM()
	default:
		logerror.Println("Invalid input bro...")
	}
}

func StartCM(IpAddress string) {
	// If cm.json is non-existent, create new CM and append to cm.json
	if _, err := os.Stat(CMPATH); os.IsNotExist(err) {
		cm := CentralManager{
			IP:        IpAddress,
			MetaData:  map[string]PageInfo{},
			IsPrimary: true,
		}

		if err := writeCMToFile([]CentralManager{cm}); err != nil {
			logerror.Println("Could not write new CM to file: ", err)
			return
		}
		logsystem.Println("Created CM and set as primary: ", cm)
		RunCM(cm)

	} else {
		// If cm.json exists, create backup CM and append to cm.json
		fileContent, err := os.ReadFile(CMPATH)
		if err != nil {
			logerror.Println("Could not read from CMPATH: ", err)
			return
		}

		// Read existing []CM
		var existingCMs []CentralManager
		if err := json.Unmarshal(fileContent, &existingCMs); err != nil {
			logerror.Println(err)
			return
		}

		// Create a backup CM and append it to the existing []CM
		backupCM := CentralManager{
			IP:        IpAddress,
			IsPrimary: false,
		}
		existingCMs = append(existingCMs, backupCM)

		// Write the updated []CM to cm.json
		if err := writeCMToFile(existingCMs); err != nil {
			logerror.Println("Could not write to CMPATH: ", err)
			return
		}
		logsystem.Println("Created Backup CM: ", backupCM)
		RunCM(backupCM)
	}
}

func RestartPrimaryCM() {
	primaryCMIP, err := getPrimaryCMIP()
	if err != nil {
		logerror.Println("Couldn't get primary CM IP: ", err)
		return
	}

	restartedCM := CentralManager{
		IP:        primaryCMIP,
		IsPrimary: true,
		MetaData:  map[string]PageInfo{},
	}

	// Ask other CM if it is primary, if so ask it to give back primary status
	allCMs := getAllCMs()
	imBack := Message{
		Type: IM_BACK,
		Payload: Payload{
			ImBack: ImBack{
				CMIP: restartedCM.IP,
			},
		},
	}

	for _, cm := range allCMs {
		reply := restartedCM.CallRPC(imBack, CENTRALMANAGER, -1, cm.IP)
		if reply.Ack {
			logsystem.Printf("Primary CM [%s] reclaiming Primary title\n", restartedCM.IP)
			restartedCM.MetaData = reply.Payload
			logsystem.Println("MetaData has been restored")
			// Get all clients to inform change of CM
			allClients := getAllClients()

			for _, client := range allClients {
				changeCM := Message{
					Type: CHANGE_CM,
					Payload: Payload{
						ChangeCM: ChangeCM{
							NewCMIP: restartedCM.IP,
						},
					},
				}
				restartedCM.CallRPC(changeCM, CLIENT, client.ID, client.IP)
			}
		}
	}

	RunCM(restartedCM)
}

func RestartBackupCM() {
	backupCMIP, err := getBackupCMIP()
	if err != nil {
		logerror.Println("Couldn't get backup CM IP: ", err)
		return
	}

	restartedBackupCM := CentralManager{
		IP:        backupCMIP,
		IsPrimary: false,
		MetaData:  map[string]PageInfo{},
	}

	RunCM(restartedBackupCM)

}

func StartClient(IpAddress string) {

	var client Client
	// If client.json is non-existent, create new Client with ID 1.
	if _, err := os.Stat(CLIENTPATH); os.IsNotExist(err) {
		// Create new client
		cmip, err := getPrimaryCMIP()
		if err != nil {
			logerror.Println("Couldn't get primary CM IP: ", err)
			return
		}
		client = Client{
			ID:        1,
			IP:        IpAddress,
			PageStore: make(map[string]Page),
			CMIP:      cmip,
		}
		if err := writeClientToFile([]Client{client}); err != nil {
			logerror.Println("Could not write to CLIENTPATH: ", err)
			return
		}
		logsystem.Printf("Created new Client with ID %d\n", client.ID)

	} else {
		// If client.json exists, read ID of latest Client (highest ID) and set ID of new client as ID+1
		fileContent, err := os.ReadFile(CLIENTPATH)
		if err != nil {
			logerror.Println("Could not read from CLIENTPATH: ", err)
			return
		}
		var existingClients []Client
		if err := json.Unmarshal(fileContent, &existingClients); err != nil {
			logerror.Println("Error Unmarshalling []Client: ", err)
		}
		// Find the highest client ID
		highestID := getHighestClientID(existingClients)

		// Create a new Client and append it to the existing Clients
		cmip, err := getPrimaryCMIP()
		if err != nil {
			logerror.Println("Couldn't get primary CM IP: ", err)
			return
		}
		client = Client{
			ID:        highestID + 1,
			IP:        IpAddress,
			PageStore: make(map[string]Page),
			CMIP:      cmip,
		}
		existingClients = append(existingClients, client)

		// Write the updated Clients to the file
		if err := writeClientToFile(existingClients); err != nil {
			logerror.Println("Could not write to CLIENTPATH: ", err)
			return
		}
		logsystem.Printf("Created new Client with ID %d\n", client.ID)
	}

	RunClient(client)
}

func RunCM(cm CentralManager) {
	// Bind yourself to a port and listen to it
	tcpAddr, err := net.ResolveTCPAddr("tcp", cm.IP)
	if err != nil {
		logerror.Println("Error resolving TCP address")
		return
	}
	inbound, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		logerror.Println("Could not listen to TCP address")
		return
	}
	// Register RPC methods and accept incoming requests
	err = rpc.Register(&cm)
	if err != nil {
		logerror.Println("Error registering CM's RPC methods: ", err)
		return
	}
	logsystem.Printf("CM is running at IP address: %s...\n", cm.IP)
	go rpc.Accept(inbound)

	if !cm.IsPrimary {
		go cm.pulseCheck()
	}
	reader := bufio.NewReader(os.Stdin)

	for {
		// Print a prompt
		logsystem.Print("> ")

		// read input from user
		input, err := reader.ReadString('\n')
		if err != nil {
			logerror.Fprintln(os.Stderr, "Error reading input:", err)
		}

		// Parse the input to handle commands
		cm.handleCMInput(strings.TrimSpace(input))
	}
}

func RunClient(c Client) {
	// Bind yourself to a port and listen to it
	tcpAddr, err := net.ResolveTCPAddr("tcp", c.IP)
	if err != nil {
		logerror.Println("Error resolving TCP address")
	}
	inbound, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		logerror.Println("Could not listen to TCP address")
	}
	// Register RPC methods and accept incoming requests
	rpc.Register(&c)
	logsystem.Printf("Client %d is running at IP address: %s...\n", c.ID, c.IP)
	go rpc.Accept(inbound)

	reader := bufio.NewReader(os.Stdin)

	for {
		// Print a prompt
		logsystem.Print("> ")

		// read input from user
		input, err := reader.ReadString('\n')
		if err != nil {
			logerror.Fprintln(os.Stderr, "Error reading input:", err)
		}

		// Parse the input to handle commands
		c.handleClientInput(strings.TrimSpace(input))
	}
}

func (cm *CentralManager) handleCMInput(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := parts[0]
	// parameters := parts[1:]

	switch command {
	case "print":
		logsystem.Println("Printing MetaData...")
		logsystem.Println(cm.MetaData)
	default:
		logsystem.Println("Invalid input brother...")
	}
}

func (c *Client) handleClientInput(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := parts[0]
	parameters := parts[1:]

	switch command {
	case "readPage":
		if len(parameters) != 1 {
			logerror.Println("Usage: readPage <pageNo>")
			return
		}
		pageNo := parameters[0]
		c.sendReadRequest(pageNo)

	case "writePage":
		if len(parameters) != 2 {
			logerror.Println("Usage: writePage <pageNo> <content>")
			return
		}
		pageNo := parameters[0]
		content := parameters[1]
		c.sendWriteRequest(pageNo, content)

	case "print":
		logsystem.Println("Printing PageStore...")
		logsystem.Println(c.PageStore)

	case "seed":
		c.seedPages()

	case "x":
		// Give some time to key in 'x' on all N terminals
		time.Sleep(30 * time.Second)
		start := time.Now().UnixMilli()
		c.randomizeRWRequests()
		end := time.Now().UnixMilli()
		timeTaken := end - start
		logsystem.Printf("TIME TAKEN: %v\n", timeTaken)
	default:
		logsystem.Println("Invalid input brother...")
	}

}
