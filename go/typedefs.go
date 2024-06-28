package main

import "database/sql"

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
}
