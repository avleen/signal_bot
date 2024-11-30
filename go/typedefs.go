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

type AppContext struct {
	DbQueryChan        chan dbQuery
	DbReplySummaryChan chan interface{}
	DbReplyAskChan     chan interface{}
	Recipients         []string
	MessagePoster      func(string, string)
	TraceContext       context.Context
}

type TimeCountCalculator struct {
	StartTime int
	Count     int
}
