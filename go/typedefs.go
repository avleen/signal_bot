package main

import (
	"context"
	"database/sql"
)

type dbQuery struct {
	query     string
	values    []interface{}
	replyChan chan dbReply
}

type dbReply struct {
	rows *sql.Rows
	err  error
}

// ImageAnalysisFunc is a function type for image analysis providers
type ImageAnalysisFunc func(attachmentId string) (string, error)

type AppContext struct {
	DbQueryChan        chan dbQuery
	DbReplySummaryChan chan interface{}
	DbReplyAskChan     chan interface{}
	Recipients         []string
	MessagePoster      func(string, string)
	TraceContext       context.Context
	ImageAnalyzer      ImageAnalysisFunc
}

type TimeCountCalculator struct {
	StartTime int
	Count     int
}

type GroupInfo struct {
	GroupID string `json:"groupId"`
}

type SentMessage struct {
	Message   string    `json:"message"`
	GroupInfo GroupInfo `json:"groupInfo"`
}

type SyncMessage struct {
	SentMessage SentMessage `json:"sentMessage"`
}

type Envelope struct {
	SourceNumber string      `json:"sourceNumber"`
	SourceName   string      `json:"sourceName"`
	Timestamp    int64       `json:"timestamp"`
	SyncMessage  SyncMessage `json:"syncMessage"`
}

type ChatbotMessage struct {
	Envelope Envelope `json:"envelope"`
}
