package main

import "time"

type CentralManager struct {
	IP        string
	MetaData  map[string]PageInfo
	IsPrimary bool
}

type PageInfo struct {
	Owner   ClientPointer
	CopySet []ClientPointer
}

func (cm *CentralManager) HandleIncomingMessage(msg Message, reply *Reply) error {
	logincoming.Printf("Message of Type [%s] received\n", msg.Type)
	if cm.IsPrimary {
		switch msg.Type {
		case READ_REQUEST:
			cm.handleReadRequest(msg)
			reply.Ack = true
		case READ_CONFIRMATION:
			cm.handleReadConfirmation(msg)
			reply.Ack = true
		case WRITE_REQUEST:
			cm.handleWriteRequest(msg)
			reply.Ack = true
		case WRITE_CONFIRMATION:
			cm.handleWriteConfirmation(msg)
			reply.Ack = true
		case PULSE:
			reply.Payload = cm.MetaData
			reply.Ack = true
		case IM_BACK:
			cm.IsPrimary = false
			reply.Payload = cm.MetaData
			reply.Ack = true
			go cm.pulseCheck()
		}
	}

	return nil
}

// Sends ReadForward to PageOwner
func (cm *CentralManager) handleReadRequest(msg Message) {
	// Check if page exists
	pageNo := msg.Payload.ReadRequest.PageNo
	page, exists := cm.MetaData[pageNo]
	if !exists {
		logerror.Printf("Page %s does not exist in CM\n", pageNo)
		logerror.Printf("ReadRequest by Client %d denied\n", msg.FromID)
		return
	}
	pageOwner := page.Owner

	// construct ReadForward message
	readForward := Message{
		Type: READ_FORWARD,
		Payload: Payload{
			ReadForward: ReadForward{
				ReadRequesterID: msg.FromID,
				ReadRequesterIP: msg.FromIP,
				PageNo:          msg.Payload.ReadRequest.PageNo,
			}},
	}

	// Send page owner ReadForward
	logoutgoing.Printf("CM sending Msg %s to Client %d\n", READ_FORWARD, pageOwner.ID)
	reply := cm.CallRPC(readForward, CLIENT, pageOwner.ID, pageOwner.IP)
	if !reply.Ack {
		logerror.Printf("Msg [%s] from CM not acknowledged\n", readForward.Type)
		return
	}
}

// Updates CopySet for Page
func (cm *CentralManager) handleReadConfirmation(msg Message) {
	requestedPage := msg.Payload.ReadConfirmation.PageNumber
	readRequesterID := msg.Payload.ReadConfirmation.ReadRequesterID
	readRequesterIP := msg.Payload.ReadConfirmation.ReadRequesterIP

	// TODO assert senderID == requestedPage.Owner
	// senderID := msg.Payload.ReadConfirmation.SenderID

	// Add requester to copyset. Update PageInfo
	pageOwner := cm.MetaData[requestedPage].Owner
	requesterPointer := ClientPointer{ID: readRequesterID, IP: readRequesterIP}
	updatedCopySet := append(cm.MetaData[requestedPage].CopySet, requesterPointer)
	cm.MetaData[requestedPage] = PageInfo{Owner: pageOwner, CopySet: updatedCopySet}
	logsystem.Println("CM updated CopySet after receiving ReadConfirmation: ", updatedCopySet)
}

// 1. Sends InvalidateCopy to clients in CopySet
// 2. Returns if any client did not ACK InvalidateCopy
// 3. If all InvalidateCopy ACKs received, send WriteForward to PageOwner
func (cm *CentralManager) handleWriteRequest(msg Message) {

	// Extract WRITEREQUEST msg info
	targetPageNo := msg.Payload.WriteRequest.PageNo
	content := msg.Payload.WriteRequest.Content
	writeRequesterID := msg.FromID
	writeRequesterIP := msg.FromIP
	writeRequesterPointer := ClientPointer{
		ID: writeRequesterID,
		IP: writeRequesterIP,
	}

	pageInfo, exists := cm.MetaData[targetPageNo]
	// If page doesnt exist (ie first time writing this page), add it to MetaData and page back to writeRequester.
	if !exists {
		logwarning.Printf("Page %s doesn't exist in CM records\n", targetPageNo)
		logwarning.Printf("Adding Page %s info to CM records...\n", targetPageNo)
		cm.MetaData[targetPageNo] = PageInfo{
			Owner:   writeRequesterPointer,
			CopySet: []ClientPointer{},
		}
		logsystem.Printf("PageInfo stored:\n%v\n", cm.MetaData[targetPageNo])

		pageSend := Message{
			Type: PAGE_SEND,
			Payload: Payload{
				PageSend: PageSend{
					Purpose: WRITE,
					Page: Page{
						Number:  targetPageNo,
						Content: content,
					},
				},
			},
		}
		reply := cm.CallRPC(pageSend, CLIENT, writeRequesterID, writeRequesterIP)
		if !reply.Ack {
			logerror.Printf("Msg [%s] from CM not acknowledged by Client %d\n", PAGE_SEND, writeRequesterID)
		}
		return
	}

	for _, clientPointer := range pageInfo.CopySet {
		invalidateCopy := Message{
			Type: INVALIDATE_COPY,
			Payload: Payload{
				InvalidateCopy: InvalidateCopy{
					WriteRequesterID: writeRequesterID,
					PageNumber:       targetPageNo,
				},
			},
		}

		reply := cm.CallRPC(invalidateCopy, CLIENT, clientPointer.ID, clientPointer.IP)
		if !reply.Ack {
			logerror.Printf("Msg [%s] from CM not acknowledged by Client %d\n", invalidateCopy.Type, clientPointer.ID)
			logerror.Println("Cannot forward Write Request")
			return
		}
	}

	// All InvalidateCopy responses have been received.
	// Send WriteForward to Page Owner
	writeForward := Message{
		Type: WRITE_FORWARD,
		Payload: Payload{
			WriteForward: WriteForward{
				WriteRequesterID: writeRequesterID,
				WriteRequesterIP: writeRequesterIP,
				PageNumber:       targetPageNo,
				Content:          content,
			},
		},
	}
	updatedPageInfo := cm.MetaData[targetPageNo]
	ownerID := updatedPageInfo.Owner.ID
	ownerIP := updatedPageInfo.Owner.IP
	reply := cm.CallRPC(writeForward, CLIENT, ownerID, ownerIP)
	if !reply.Ack {
		logerror.Printf("Msg [%s] from CM not acknowledged by Client %d\n", writeForward.Type, ownerID)
		return
	}
}

func (cm *CentralManager) handleWriteConfirmation(msg Message) {
	// change owner of page to sender of writeConfirmation.
	// make sure copyset is null until other reads come in

	newlyWrittenPageNo := msg.Payload.WriteConfirmation.PageNumber
	newlyWrittenPage, exists := cm.MetaData[newlyWrittenPageNo]
	if !exists {
		logerror.Printf("CM does not have PageInfo of Page %s", newlyWrittenPageNo)
		return
	}

	writerID := msg.Payload.WriteConfirmation.WriterID
	writerIP := msg.Payload.WriteConfirmation.WriterIP

	// Update Owner of page and clear CopySet
	newlyWrittenPage.Owner = ClientPointer{ID: writerID, IP: writerIP}
	newlyWrittenPage.CopySet = []ClientPointer{}
	cm.MetaData[newlyWrittenPageNo] = newlyWrittenPage
}

func (cm *CentralManager) pulseCheck() {
	for {
		time.Sleep(2 * time.Second)

		pulse := Message{
			Type: PULSE,
			Payload: Payload{
				Pulse: Pulse{
					FromIP: cm.IP,
				},
			},
		}
		primaryCMIP, err := getPrimaryCMIP()
		if err != nil {
			logerror.Println("Backup CM could not get primary CMIP")
			return
		}
		reply := cm.CallRPC(pulse, CENTRALMANAGER, -1, primaryCMIP)
		if !reply.Ack {
			logerror.Println("PULSE not returned by Primary CM")
			logerror.Println("Primary CM is likely dead!!")
			logsystem.Println("It's time...")
			logsystem.Println("Backup CM undergoing transformation...")
			cm.IsPrimary = true
			logsystem.Println("Backup CM is now Primary CM!!!")

			clientArr := getAllClients()
			for _, client := range clientArr {
				changeCM := Message{
					Type: CHANGE_CM,
					Payload: Payload{
						ChangeCM: ChangeCM{
							NewCMIP: cm.IP,
						},
					},
				}
				reply := cm.CallRPC(changeCM, CLIENT, client.ID, client.IP)
				if !reply.Ack {
					logerror.Printf("Msg [%s] from CM not acknowledged by Client %d\n", CHANGE_CM, client.ID)
					return
				}
			}
			return
		} else {
			cm.MetaData = reply.Payload
		}
	}
}
