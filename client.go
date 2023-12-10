package main

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	READ      = "READ"
	WRITE     = "WRITE"
	READWRITE = "READWRITE"
	NIL       = "NIL"
)

type Client struct {
	ID        int
	IP        string
	PageStore map[string]Page
	CMIP      string
}

type ClientPointer struct {
	ID int
	IP string
}

type Page struct {
	Number  string
	Content string
	Access  string
}

func (c *Client) HandleIncomingMessage(msg Message, reply *Reply) error {
	logincoming.Printf("Message of Type [%s] received\n", msg.Type)
	switch msg.Type {
	case READ_FORWARD:
		c.handleReadForward(msg)
		reply.Ack = true
	case PAGE_SEND:
		c.handlePageSend(msg)
		reply.Ack = true
	case INVALIDATE_COPY:
		reply.Ack = c.handleInvalidateCopy(msg)
	case WRITE_FORWARD:
		c.handleWriteForward(msg)
		reply.Ack = true
	case CHANGE_CM:
		c.handleChangeCM(msg)
		reply.Ack = true
	}
	return nil
}

// Sends PageSend to ReadRequester
func (c *Client) handleReadForward(msg Message) {
	// Construct PageSend message
	requestedPageNo := msg.Payload.ReadForward.PageNo
	requestedPage := c.PageStore[requestedPageNo]
	pageSendMsg := Message{
		Type: PAGE_SEND,
		Payload: Payload{
			PageSend: PageSend{
				Purpose: READ,
				Page:    requestedPage,
			}},
		FromID: c.ID,
		FromIP: c.IP,
	}

	readRequestedID := msg.Payload.ReadForward.ReadRequesterID
	readRequesterIP := msg.Payload.ReadForward.ReadRequesterIP

	logoutgoing.Printf("Client %d sending Msg %s to Client %d\n", c.ID, PAGE_SEND, readRequestedID)
	reply := c.CallRPC(pageSendMsg, CLIENT, readRequestedID, readRequesterIP)
	if !reply.Ack {
		logerror.Printf("Msg [%s] from Client %d not acknowledged by Client %d\n", pageSendMsg.Type, c.ID, readRequestedID)
		return
	}
}

// Replaces old page with new page in PageSend.
func (c *Client) handlePageSend(msg Message) {
	// Add page to PageStore
	sentPageNo := msg.Payload.PageSend.Page.Number
	sentPage := msg.Payload.PageSend.Page
	purpose := msg.Payload.PageSend.Purpose

	if purpose == READ {
		sentPage.Access = READ
		readConf := Message{
			Type: READ_CONFIRMATION,
			Payload: Payload{
				ReadConfirmation: ReadConfirmation{
					PageNumber:      sentPageNo,
					ReadRequesterID: c.ID,
					ReadRequesterIP: c.IP,
					SenderID:        msg.FromID,
					SenderIP:        msg.FromIP,
				},
			},
		}
		reply := c.CallRPC(readConf, CENTRALMANAGER, -1, c.CMIP)
		if !reply.Ack {
			logerror.Printf("Msg [%s] from Client %d not acknowledged by CM\n", READ_CONFIRMATION, c.ID)
			return
		}

	} else if purpose == WRITE {
		sentPage.Access = READWRITE

		writeConf := Message{
			Type: WRITE_CONFIRMATION,
			Payload: Payload{
				WriteConfirmation: WriteConfirmation{
					PageNumber: sentPageNo,
					WriterID:   c.ID,
					WriterIP:   c.IP,
				},
			},
		}
		reply := c.CallRPC(writeConf, CENTRALMANAGER, -1, c.CMIP)
		if !reply.Ack {
			logerror.Printf("Msg [%s] from Client %d not acknowledged by CM\n", WRITE_CONFIRMATION, c.ID)
			return
		}
	}

	c.PageStore[sentPageNo] = sentPage
}

// Sets targetPage.Access as NIL
func (c *Client) handleInvalidateCopy(msg Message) bool {
	targetPageNo := msg.Payload.InvalidateCopy.PageNumber
	targetPage, exists := c.PageStore[targetPageNo]
	if !exists {
		logerror.Printf("Page %s doesn't exist in Node %d's PageStore. Cannot invalidate", targetPageNo, c.ID)
		return false
	}

	targetPage.Access = NIL
	c.PageStore[targetPageNo] = targetPage
	return true
}

// Sets own Page.Access to NIL
// Sends Page to writeRequester
func (c *Client) handleWriteForward(msg Message) {
	// Extract WRITEFORWARD msg content
	writeRequesterID := msg.Payload.WriteForward.WriteRequesterID
	writeRequesterIP := msg.Payload.WriteForward.WriteRequesterIP
	requestedPage := msg.Payload.WriteForward.PageNumber
	content := msg.Payload.WriteForward.Content

	// Get page from PageStore, set access to NIL, update content.
	page, exists := c.PageStore[requestedPage]
	page.Access = NIL
	page.Content = content
	c.PageStore[requestedPage] = page

	if !exists {
		logerror.Printf("Page %s requested (to write) by Client %d does not exist in Client %d's PageStore", requestedPage, writeRequesterID, c.ID)
		return
	}

	pageSend := Message{
		Type: PAGE_SEND,
		Payload: Payload{
			PageSend: PageSend{
				Purpose: WRITE,
				Page:    page,
			},
		},
		FromID: c.ID,
		FromIP: c.IP,
	}
	logoutgoing.Printf("Client %d sending Msg %s to Client %d\n", c.ID, PAGE_SEND, writeRequesterID)
	reply := c.CallRPC(pageSend, CLIENT, writeRequesterID, writeRequesterIP)
	if !reply.Ack {
		logerror.Printf("Msg [%s] from Client %d not acknowledged by Client %d\n", INVALIDATE_CONFIRMATION, c.ID, writeRequesterID)
		return
	}
}

func (c *Client) sendReadRequest(pageNo string) {
	readRequest := Message{
		Type: READ_REQUEST,
		Payload: Payload{
			ReadRequest: ReadRequest{
				PageNo: pageNo,
			},
		},
		FromID: c.ID,
		FromIP: c.IP,
	}

	reply := c.CallRPC(readRequest, CENTRALMANAGER, -1, c.CMIP)
	if !reply.Ack {
		logerror.Printf("Msg [%s] from Client %d not acknowledged by CM\n", READ_REQUEST, c.ID)
	}
}

func (c *Client) sendWriteRequest(pageNo string, content string) {
	page, exists := c.PageStore[pageNo]
	if exists {
		if page.Access == READWRITE {
			logsystem.Printf("Page %s exists in local storage with %s access\n", pageNo, page.Access)
			logsystem.Printf("Writing new content in local page...\n")
			page.Content = content
			c.PageStore[pageNo] = page
			return
		} else {
			logsystem.Printf("Page %s exists in local storage with %s access\n", pageNo, page.Access)
			logsystem.Println("Page Fault...")
		}
	} else {
		logsystem.Printf("Page %s does not exist in local storage\n", pageNo)
		logsystem.Println("Page Fault...")
	}

	writeRequest := Message{
		Type: WRITE_REQUEST,
		Payload: Payload{
			WriteRequest: WriteRequest{
				PageNo:  pageNo,
				Content: content,
			},
		},
		FromID: c.ID,
		FromIP: c.IP,
	}

	reply := c.CallRPC(writeRequest, CENTRALMANAGER, -1, c.CMIP)
	if !reply.Ack {
		logerror.Printf("Msg [%s] from Client %d not acknowledged by CM\n", READ_REQUEST, c.ID)
	}
}

func (c *Client) handleChangeCM(msg Message) {
	c.CMIP = msg.Payload.ChangeCM.NewCMIP
	logsystem.Printf("Changed CMIP to %s\n", c.CMIP)
}

func (c *Client) seedPages() {
	for i := 1; i <= 10; i++ {
		c.sendWriteRequest(fmt.Sprintf("P%d", i), fmt.Sprintf("Content by Client %d", c.ID))
	}
}

func (c *Client) randomizeRWRequests() {

	for i := 0; i < NUM_REQS; i++ {
		time.Sleep(1 * time.Second)
		randomNumber := rand.Intn(2)
		if randomNumber == 0 {
			c.sendWriteRequest(fmt.Sprintf("P%d", rand.Intn(10)), fmt.Sprintf("Content by Client %d", c.ID))
		} else {
			c.sendReadRequest(fmt.Sprintf("P%d", rand.Intn(10)))
		}
	}
}
