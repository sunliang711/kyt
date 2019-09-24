package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/sunliang711/kyt/models"
	"github.com/sunliang711/kyt/utils"
)

type tx struct {
	ID  int64  `json:"id"`
	Tag string `json:"tag"`
}

func blockTxList(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	var (
		err         error
		blockheight int
		rows        *sql.Rows
		txID        int64
		tag         string
	)
	blockid := query.Get("blockid")
	blockhash := query.Get("blockhash")
	if len(blockid) != 0 {
		blockheight, err = strconv.Atoi(blockid)
		if err != nil {
			utils.JsonResponse(resp{1, "block id format error", nil}, w)
			return
		}
		log.Printf("Got blockheight: %v\n", blockheight)

	} else if len(blockhash) != 0 {
		log.Printf("Got blockhash: %v", blockhash)
		//如果是blockhash,要通过blockhash拿到blockid
		rows, err := models.DB.Query("select blockID from blockinfo where blockhash=?", blockhash)
		if err != nil {
			utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select blockID from blockinfo where blockhash = %v", blockhash), nil}, w)
			return
		}
		if rows.Next() {
			rows.Scan(&blockheight)
			rows.Close()
		} else {
			utils.JsonResponse(resp{1, fmt.Sprintf("no record in blockinfo where blockhash = %v", blockhash), nil}, w)
			return
		}
		log.Printf("Got blockheight: %v\n", blockhash)

	} else {
		utils.JsonResponse(resp{1, "Please give me blockid or blockhash parameter!", nil}, w)
		return
	}

	//blockheight got
	var txIDs []int64
	//根据blockid，到txinfo表拿txID
	rows, err = models.DB.Query("select txID from txinfo where blockID = ?", blockheight)
	if err != nil {
		utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select txID from txinfo where blockID = %v", blockheight), nil}, w)
		return
	}
	for rows.Next() {
		rows.Scan(&txID)
		txIDs = append(txIDs, txID)
	}
	rows.Close()
	log.Printf("txIDs: %v\n", txIDs)
	params := []interface{}{}
	for _, v := range txIDs {
		params = append(params, v)
	}
	rows, err = models.DB.Query("select txID,risktag from txhash where txID in "+utils.MakeQuestion(len(txIDs)), params...)
	if err != nil {
		utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select txID,risktag from txhash where txID in %v", txIDs), nil}, w)
		return
	}
	var txs []tx
	for rows.Next() {
		rows.Scan(&txID, &tag)
		txs = append(txs, tx{txID, tag})
	}
	rows.Close()

	utils.JsonResponse(resp{0, "OK", txs}, w)

}

// blockTransactionGraph TODO
// 2019/09/24 15:56:39
// 找出一个块中所有交易，这些交易的srcs到dests的图
func blockTransactionGraph(w http.ResponseWriter, req http.Request) {
	query := req.URL.Query()
	var (
		hash   string
		height int
		err    error
	)
	hash = query.Get("blockhash")
	if hash == "" {
		heightStr := query.Get("blockheight")
		height, err = strconv.Atoi(heightStr)
		if err != nil {
			utils.JsonResponse(resp{1, "blockHeight invalid", nil}, w)
			return
		}
		fmt.Println(height)
	}
}

// addressNode TODO
// 2019/09/24 19:03:57
type addressNode struct {
	ID          int    `json:"id"`          //addrID
	User        int    `json:"user"`        //userID
	Description string `json:"description"` //addrHash
}

// addressLink TODO
// 2019/09/24 19:06:54
type addressLink struct {
	Source string `json:"source'`
	Target string `json:"target'`
}

// oneTxGraph TODO
// 2019/09/24 19:01:38
func oneTxGraph(txID int) (addressNodes []addressNode, addressLinks []addressLink, err error) {
	// txin table
	// sql1 := "select a.addrID,b.address from txin a join addresses b on (a.addrID=b.addrID) where a.txID = ?"
	// sql2 := "select a.addrID,b.userID from txin a join entities b on (a.addrID=b.addrID) where a.txID = ?"
	// sql1 + sql2 => sql

	//tx sources
	sql := "select c.addrID,c.address,d.userID from entities d join (select a.addrID,b.address from txin a join addresses b on (a.addrID=b.addrID) where a.txID = ?) c on (d.addrID=c.addrID)"
	rows, err := models.DB.Query(sql, txID)
	if err != nil {
		log.Printf("execute sql: %v error: %v", sql, err)
		return nil, nil, err
	}
	for rows.Next() {
		var (
			addrID  int
			address string
			userID  int
		)
		rows.Scan(&addrID, &address, &userID)
		addressNodes = append(addressNodes, addressNode{addrID, userID, address})
	}

	// txout table
	// tx dests
	sql = "select c.addrID,c.address,d.userID from entities d join (select a.addrID,b.address from txout a join addresses b on (a.addrID=b.addrID) where a.txID = ?) c on (d.addrID=c.addrID)"
	rows, err = models.DB.Query(sql, txID)
	if err != nil {
		log.Printf("execute sql: %v error: %v", sql, err)
		return nil, nil, err
	}

	for rows.Next() {
		var (
			addrID  int
			address string
			userID  int
		)
		rows.Scan(&addrID, &address, &userID)
		addressNodes = append(addressNodes, addressNode{addrID, userID, address})
	}

	//link = srcs => dests
}
