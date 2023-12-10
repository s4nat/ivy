package main

const (
	READ_REQUEST            = "READ_REQUEST"
	READ_FORWARD            = "READ_FORWARD"
	PAGE_SEND               = "PAGE_SEND"
	READ_CONFIRMATION       = "READ_CONFIRMATION"
	WRITE_REQUEST           = "WRITE_REQUEST"
	INVALIDATE_COPY         = "INVALIDATE_COPY"
	INVALIDATE_CONFIRMATION = "INVALIDATE_CONFIRMATION"
	WRITE_FORWARD           = "WRITE_FORWARD"
	WRITE_CONFIRMATION      = "WRITE_CONFIRMATION"
	PULSE                   = "PULSE"
	CHANGE_CM               = "CHANGE_CM"
	IM_BACK                 = "IM_BACK"
)

type Message struct {
	Type    string
	Payload Payload
	FromID  int
	FromIP  string
}

type Reply struct {
	Ack     bool
	Payload map[string]PageInfo
}

type Payload struct {
	ReadRequest            ReadRequest
	ReadForward            ReadForward
	PageSend               PageSend
	ReadConfirmation       ReadConfirmation
	WriteRequest           WriteRequest
	InvalidateCopy         InvalidateCopy
	InvalidateConfirmation InvalidateConfirmation
	WriteForward           WriteForward
	WriteConfirmation      WriteConfirmation
	Pulse                  Pulse
	ChangeCM               ChangeCM
	ImBack                 ImBack
}

type ReadRequest struct {
	PageNo string
}

type ReadForward struct {
	ReadRequesterID int
	ReadRequesterIP string
	PageNo          string
}

type PageSend struct {
	Purpose string
	Page    Page
}

type ReadConfirmation struct {
	PageNumber      string
	ReadRequesterID int
	ReadRequesterIP string
	SenderID        int
	SenderIP        string
}

type WriteRequest struct {
	PageNo  string
	Content string
}

type InvalidateCopy struct {
	WriteRequesterID int
	PageNumber       string
}

type InvalidateConfirmation struct {
	WriteRequesterID int
	PageNumber       string
}

type WriteForward struct {
	WriteRequesterID int
	WriteRequesterIP string
	PageNumber       string
	Content          string
}

type WriteConfirmation struct {
	WriterID   int
	WriterIP   string
	PageNumber string
}

type Pulse struct {
	FromIP string
}

type ChangeCM struct {
	NewCMIP string
}

type ImBack struct {
	CMIP string
}
