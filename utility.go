package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
)

const (
	CLIENT         = "Client"
	CENTRALMANAGER = "CentralManager"
)

func (cm *CentralManager) CallRPC(msg Message, nodeType string, targetID int, targetIP string) (reply Reply) {
	logoutgoing.Printf("CM with IP: %s is sending message %s to Client [%d] with IP: %s\n", cm.IP, msg.Type, targetID, targetIP)
	clnt, err := rpc.Dial("tcp", targetIP)
	if err != nil {
		logerror.Println("Error dialing RPC: ", err)
		reply.Ack = false
		return reply
	}
	err = clnt.Call(fmt.Sprintf("%s.HandleIncomingMessage", nodeType), msg, &reply)
	if err != nil {
		logerror.Printf("Error calling RPC from Msg [%s]: %v\n", msg.Type, err)
		reply.Ack = false
		return reply
	}
	return reply
}

func (client *Client) CallRPC(msg Message, nodeType string, targetID int, targetIP string) (reply Reply) {
	logoutgoing.Printf("Client [%d] with IP: [%s] is sending message %s to %s [%d] with IP [%s]\n", client.ID, client.IP, msg.Type, nodeType, targetID, targetIP)
	clnt, err := rpc.Dial("tcp", targetIP)
	if err != nil {
		logerror.Println("Error dialing RPC: ", err)
		reply.Ack = false
		return reply
	}
	err = clnt.Call(fmt.Sprintf("%s.HandleIncomingMessage", nodeType), msg, &reply)
	if err != nil {
		logerror.Printf("Error calling RPC from Msg [%s]: %v\n", msg.Type, err)
		reply.Ack = false
		return reply
	}
	return reply
}

/*
Function to automatically get the outbound IP without user input in .env file
*/
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

/*
Function to get a port number that is currently not in use
*/
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func writeCMToFile(cms []CentralManager) error {
	// Serialize CM to JSON
	cmJSON, err := json.MarshalIndent(cms, "", "  ")
	if err != nil {
		return err
	}

	// Write JSON to cm.json file
	err = os.WriteFile(CMPATH, cmJSON, 0644)
	if err != nil {
		return err
	}

	return nil
}

func writeClientToFile(clients []Client) error {
	// Serialize Clients to JSON
	clientJSON, err := json.MarshalIndent(clients, "", "  ")
	if err != nil {
		return err
	}

	// Write JSON to client.json file
	err = os.WriteFile(CLIENTPATH, clientJSON, 0644)
	if err != nil {
		return err
	}

	return nil
}

func getPrimaryCMIP() (string, error) {
	// Read cm.json to get the IP of the primary CM
	fileContent, err := os.ReadFile(CMPATH)
	if err != nil {
		logerror.Println("Error reading cm.json: ", err)
		return "NIL", err // Handle error accordingly
	}

	var cms []CentralManager
	if err := json.Unmarshal(fileContent, &cms); err != nil {
		logerror.Println("Error Unmarshalling []CM: ", err)
	}

	// Find the primary CM and return its IP
	for _, cm := range cms {
		if cm.IsPrimary {
			return cm.IP, nil
		}
	}

	logerror.Println("No Primary CM found: ", err)
	return "NIL", err
}

func getBackupCMIP() (string, error) {
	// Read cm.json to get the IP of the primary CM
	fileContent, err := os.ReadFile(CMPATH)
	if err != nil {
		logerror.Println("Error reading cm.json: ", err)
		return "NIL", err // Handle error accordingly
	}

	var cms []CentralManager
	if err := json.Unmarshal(fileContent, &cms); err != nil {
		logerror.Println("Error Unmarshalling []CM: ", err)
	}

	// Find the primary CM and return its IP
	for _, cm := range cms {
		if !cm.IsPrimary {
			return cm.IP, nil
		}
	}

	logerror.Println("No Primary CM found: ", err)
	return "NIL", err
}

func getHighestClientID(clients []Client) int {
	if len(clients) == 0 {
		return -1
	}

	highestID := 0
	for _, client := range clients {
		if client.ID > highestID {
			highestID = client.ID
		}
	}
	return highestID
}

func getAllClients() []Client {
	// Read client.json to get all Clients
	fileContent, err := os.ReadFile(CLIENTPATH)
	if err != nil {
		logerror.Println("Error reading client.json: ", err)
		return []Client{}
	}

	var clientArr []Client
	if err := json.Unmarshal(fileContent, &clientArr); err != nil {
		logerror.Println("Error Unmarshalling []Client: ", err)
	}

	return clientArr
}

func getAllCMs() []CentralManager {
	// Read cm.json to get all CMs
	fileContent, err := os.ReadFile(CMPATH)
	if err != nil {
		logerror.Println("Error reading cm.json: ", err)
		return []CentralManager{}
	}

	var CMArr []CentralManager
	if err := json.Unmarshal(fileContent, &CMArr); err != nil {
		logerror.Println("Error Unmarshalling []CM: ", err)
	}
	return CMArr
}
